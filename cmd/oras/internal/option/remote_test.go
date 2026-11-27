/*
Copyright The ORAS Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package option

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/oras-project/oras-go/v3/registry/remote"
	"github.com/oras-project/oras-go/v3/registry/remote/auth"
	"github.com/oras-project/oras-go/v3/registry/remote/credentials"
	"github.com/oras-project/oras-go/v3/registry/remote/properties"
	"github.com/oras-project/oras-go/v3/registry/remote/retry"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var ts *httptest.Server
var testRepo = "test-repo"
var testTagList = struct {
	Tags []string `json:"tags"`
}{
	Tags: []string{"tag"},
}

// localhostServerCert is a PEM-encoded TLS cert with SAN IPs
// "127.0.0.1" and "[::1]", expiring at Jan 29 16:00:00 2084 GMT.
// adapted from golang crypto/tls:
// go run generate_cert.go  --rsa-bits 4096 --host 127.0.0.1,::1,oras.land --ca --start-date "Jan 1 00:00:00 1970" --duration=1000000h
//
//go:embed testdata/localhostServer.crt
var localhostServerCert []byte

// localhostServerKey is the private key for localhostServerCert.
//
//go:embed testdata/localhostServer.key
var localhostServerKey []byte

// localhostClientCert is a PEM-encoded TLS cert with SAN IPs
// "127.0.0.1" and "[::1]", expiring at Jan 29 16:00:00 2084 GMT.
// adapted from golang crypto/tls (added Client Auth usage):
// go run generate_cert.go  --rsa-bits 4096 --host 127.0.0.1,::1,oras.land --ca --start-date "Jan 1 00:00:00 1970" --duration=1000000h
//
//go:embed testdata/localhostClient.crt
var localhostClientCert []byte

// localhostClientKey is the private key for localhostClientCert.
//
//go:embed testdata/localhostClient.key
var localhostClientKey []byte

func testingKey(s []byte) []byte {
	return bytes.ReplaceAll(s, []byte("TESTING KEY"), []byte("PRIVATE KEY"))
}

func loadTestingTLSConfig() *tls.Config {
	clientCertPool := x509.NewCertPool()
	clientCertPool.AppendCertsFromPEM(localhostClientCert)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{loadTestingCert(localhostServerCert, testingKey(localhostServerKey))},
		ClientAuth:   tls.VerifyClientCertIfGiven,
		ClientCAs:    clientCertPool,
	}

	return tlsConfig
}

func loadTestingCert(certificate, key []byte) tls.Certificate {
	cert, err := tls.X509KeyPair(certificate, key)
	if err != nil {
		panic(fmt.Sprintf("Unable to load testing certificate: %v", err))
	}

	return cert
}

func TestMain(m *testing.M) {
	// Test server
	ts = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		m := r.Method
		switch {
		case p == "/v2/" && m == "GET":
			w.WriteHeader(http.StatusOK)
		case p == fmt.Sprintf("/v2/%s/tags/list", testRepo) && m == "GET":
			if err := json.NewEncoder(w).Encode(testTagList); err != nil {
				http.Error(w, "error encoding", http.StatusBadRequest)
			}
		}
	}))
	ts.TLS = loadTestingTLSConfig()
	ts.StartTLS()
	defer ts.Close()
	os.Exit(m.Run())
}

func TestRemote_FlagsInit(_ *testing.T) {
	var test struct {
		Remote
	}

	ApplyFlags(&test, pflag.NewFlagSet("oras-test", pflag.ExitOnError))
}

func writeTestAuthConfig(t *testing.T, auths map[string]credentials.Credential) string {
	t.Helper()
	entries := make(map[string]map[string]string, len(auths))
	for host, cred := range auths {
		encoded := base64.StdEncoding.EncodeToString([]byte(cred.Username + ":" + cred.Password))
		entries[host] = map[string]string{"auth": encoded}
	}
	data, err := json.Marshal(map[string]any{"auths": entries})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return configPath
}

func TestRemote_authClient_RawCredential(t *testing.T) {
	password := make([]byte, 12)
	if _, err := rand.Read(password); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := credentials.Credential{
		Username: "mocked^^??oras-@@!#",
		Password: base64.StdEncoding.EncodeToString(password),
	}
	opts := Remote{
		Username: want.Username,
		Secret:   want.Password,
		Configs:  []string{writeTestAuthConfig(t, nil)},
	}
	client, err := opts.authClient("hostname", false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := client.CredentialFunc(t.Context(), properties.Resource{Registry: "hostname"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Fatalf("expect: %v, got: %v", want, got)
	}

	other, err := client.CredentialFunc(t.Context(), properties.Resource{Registry: "other.example"})
	if err != nil {
		t.Fatalf("unexpected error for other host: %v", err)
	}
	if other != credentials.EmptyCredential {
		t.Fatalf("expected empty credential for other host, got: %v", other)
	}
}

func TestRemote_authClient_SharedCache(t *testing.T) {
	opts := Remote{
		Username:   "test-user",
		Secret:     "test-password",
		authCaches: newAuthCacheStore(),
	}
	first, err := opts.authClient("hostname", false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	const (
		registry = "registry.example.com"
		cacheKey = "repository:test:pull"
		want     = "test-token"
	)
	if _, err := first.Cache.Set(context.Background(), registry, auth.SchemeBearer, cacheKey, func(context.Context) (string, error) {
		return want, nil
	}); err != nil {
		t.Fatalf("unexpected error when populating shared cache: %v", err)
	}

	second, err := opts.authClient("hostname", false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if second == first {
		t.Fatal("expected a new auth client")
	}
	got, err := second.Cache.GetToken(context.Background(), registry, auth.SchemeBearer, cacheKey)
	if err != nil {
		t.Fatalf("unexpected error when reading shared cache: %v", err)
	}
	if got != want {
		t.Fatalf("expected cached token %q, got %q", want, got)
	}
}

func TestRemote_authClient_SharedCacheByCredential(t *testing.T) {
	const (
		registry = "registry.example.com"
		cacheKey = "repository:test:pull"
	)
	store := newAuthCacheStore()
	newClient := func(t *testing.T, opts Remote) *auth.Client {
		t.Helper()
		opts.authCaches = store
		client, err := opts.authClient(registry, false, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return client
	}
	from := newClient(t, Remote{Username: "user-a", Secret: "password-a"})
	if _, err := from.Cache.Set(context.Background(), registry, auth.SchemeBearer, cacheKey, func(context.Context) (string, error) {
		return "token-a", nil
	}); err != nil {
		t.Fatalf("unexpected error when populating cache: %v", err)
	}

	tests := []struct {
		name   string
		opts   Remote
		shared bool
	}{
		{name: "same credential", opts: Remote{Username: "user-a", Secret: "password-a"}, shared: true},
		{name: "different username", opts: Remote{Username: "user-b", Secret: "password-a"}},
		{name: "different password", opts: Remote{Username: "user-a", Secret: "password-b"}},
		{name: "identity token", opts: Remote{Secret: "password-a"}},
		{name: "credential store", opts: Remote{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newClient(t, tt.opts)
			got, err := client.Cache.GetToken(context.Background(), registry, auth.SchemeBearer, cacheKey)
			if tt.shared {
				if err != nil || got != "token-a" {
					t.Fatalf("expected shared cached token, got %q, err %v", got, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected no cached token, got %q", got)
			}
		})
	}
}

func TestRemote_authClient_SharedCacheByConfig(t *testing.T) {
	tmp := t.TempDir()
	configA := filepath.Join(tmp, "a.json")
	configB := filepath.Join(tmp, "b.json")
	store := newAuthCacheStore()
	newCache := func(t *testing.T, configs ...string) auth.Cache {
		t.Helper()
		opts := Remote{Configs: configs, authCaches: store}
		client, err := opts.authClient("hostname", false, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return client.Cache
	}
	if newCache(t, configA) != newCache(t, configA) {
		t.Fatal("expected clients using the same registry config to share a cache")
	}
	if newCache(t, configA) == newCache(t, configB) {
		t.Fatal("expected clients using different registry configs not to share a cache")
	}
}

func TestRemote_authClient_InvocationScopedCache(t *testing.T) {
	const (
		registry = "registry.example.com"
		cacheKey = "repository:test:pull"
	)
	newClient := func(t *testing.T, store *authCacheStore, opts Remote) *auth.Client {
		t.Helper()
		opts.authCaches = store
		client, err := opts.authClient(registry, false, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return client
	}

	// Clients built within the same invocation (same store) with the same
	// credentials share a cache.
	store := newAuthCacheStore()
	from := newClient(t, store, Remote{Username: "user-a", Secret: "password-a"})
	if _, err := from.Cache.Set(context.Background(), registry, auth.SchemeBearer, cacheKey, func(context.Context) (string, error) {
		return "token-a", nil
	}); err != nil {
		t.Fatalf("unexpected error when populating cache: %v", err)
	}
	to := newClient(t, store, Remote{Username: "user-a", Secret: "password-a"})
	got, err := to.Cache.GetToken(context.Background(), registry, auth.SchemeBearer, cacheKey)
	if err != nil || got != "token-a" {
		t.Fatalf("expected shared cache within one invocation, got %q, err %v", got, err)
	}

	// Clients built in a different invocation (different store) do not share
	// the cache, even with identical credentials.
	otherStore := newAuthCacheStore()
	other := newClient(t, otherStore, Remote{Username: "user-a", Secret: "password-a"})
	if _, err := other.Cache.GetToken(context.Background(), registry, auth.SchemeBearer, cacheKey); err == nil {
		t.Fatal("expected cache not to leak across invocations")
	}
}

func TestRemote_authClient_SharedCacheWiring(t *testing.T) {
	const registry = "registry.example.com"
	newCache := func(t *testing.T, common Common, opts Remote) auth.Cache {
		t.Helper()
		opts.plainHTTP = plainHTTPNotSpecified
		repo, err := opts.NewRepository(registry+"/test", common, logrus.New())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		client, ok := repo.Registry.Client.(*auth.Client)
		if !ok {
			t.Fatalf("unexpected client type %T", repo.Registry.Client)
		}
		return client.Cache
	}

	common := Common{authCaches: newAuthCacheStore()}
	if got, want := newCache(t, common, Remote{Username: "user-a", Secret: "password-a"}), newCache(t, common, Remote{Username: "user-a", Secret: "password-a"}); got != want {
		t.Fatal("expected clients with the same credentials in one invocation to share a cache")
	}
	if newCache(t, common, Remote{Username: "user-a", Secret: "password-a"}) == newCache(t, common, Remote{Username: "user-b", Secret: "password-b"}) {
		t.Fatal("expected clients with different credentials not to share a cache")
	}
}

func TestRemote_ensureAuthCaches_Reexecution(t *testing.T) {
	cmd := &cobra.Command{}
	common := Common{}
	remote := Remote{}
	if err := common.Parse(cmd); err != nil {
		t.Fatal(err)
	}
	remote.ensureAuthCaches(common)
	first := remote.authCaches
	if first == nil {
		t.Fatal("expected initial auth cache store")
	}
	if err := common.Parse(cmd); err != nil {
		t.Fatal(err)
	}
	remote.ensureAuthCaches(common)
	if remote.authCaches == first || remote.authCaches != common.authCaches {
		t.Fatal("expected remote to bind to the new invocation's store")
	}
	remote.ensureAuthCaches(Common{})
	if remote.authCaches != common.authCaches {
		t.Fatal("missing common store must not reset the remote cache")
	}
}

func TestRemote_authClient_skipTlsVerify(t *testing.T) {
	opts := Remote{
		Insecure: true,
	}
	client, err := opts.authClient("hostname", false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
}

func TestRemote_authClient_CARoots(t *testing.T) {
	caPath := filepath.Join(t.TempDir(), "oras-test.pem")
	if err := os.WriteFile(caPath, localhostServerCert, 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	opts := Remote{
		CACertFilePath: caPath,
	}
	client, err := opts.authClient("hostname", false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
}

func TestRemote_authClient_resolve(t *testing.T) {
	URL, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("invalid url in test server: %s", ts.URL)
	}

	testHost := "test.unit.oras"
	opts := Remote{
		resolveFlag: []string{fmt.Sprintf("%s:%s:%s", testHost, URL.Port(), URL.Hostname())},
		Insecure:    true,
	}
	client, err := opts.authClient(testHost, false, false)
	if err != nil {
		t.Fatalf("unexpected error when creating auth client: %v", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("https://%s:%s", testHost, URL.Port()), nil)
	if err != nil {
		t.Fatalf("unexpected error when generating request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error when sending request: %v", err)
	}
	resp.Body.Close()
}

func plainHTTPEnabled() (plainHTTP bool, fromFlag bool) {
	return true, true
}
func HTTPSEnabled() (plainHTTP bool, fromFlag bool) {
	return false, true
}
func plainHTTPNotSpecified() (plainHTTP bool, fromFlag bool) {
	return false, false
}

func TestRemote_NewRegistry(t *testing.T) {
	caPath := filepath.Join(t.TempDir(), "oras-test.pem")
	if err := os.WriteFile(caPath, localhostServerCert, 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	opts := struct {
		Remote
		Common
	}{
		Remote{
			CACertFilePath: caPath,
			plainHTTP:      plainHTTPNotSpecified,
		},
		Common{},
	}
	uri, err := url.ParseRequestURI(ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	reg, err := opts.NewRegistry(uri.Host, opts.Common, logrus.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err = reg.Ping(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemote_NewRepository(t *testing.T) {
	caPath := filepath.Join(t.TempDir(), "oras-test.pem")
	if err := os.WriteFile(caPath, localhostServerCert, 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	opts := struct {
		Remote
		Common
	}{
		Remote{
			CACertFilePath: caPath,
			plainHTTP:      plainHTTPNotSpecified,
		},
		Common{},
	}

	uri, err := url.ParseRequestURI(ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	repo, err := opts.NewRepository(uri.Host+"/"+testRepo, opts.Common, logrus.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err = repo.Tags(context.Background(), "", func(got []string) error {
		want := []string{"tag"}
		if len(got) != len(testTagList.Tags) || !reflect.DeepEqual(got, want) {
			return fmt.Errorf("expect: %v, got: %v", testTagList.Tags, got)
		}
		return nil
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemote_NewRepositoryMTLS(t *testing.T) {
	caPath := filepath.Join(t.TempDir(), "oras-test.pem")
	if err := os.WriteFile(caPath, localhostServerCert, 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	clientCertPath := filepath.Join(t.TempDir(), "oras-test-client.pem")
	if err := os.WriteFile(clientCertPath, localhostClientCert, 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	clientKeyPath := filepath.Join(t.TempDir(), "oras-test-client.key")
	if err := os.WriteFile(clientKeyPath, testingKey(localhostClientKey), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	opts := struct {
		Remote
		Common
	}{
		Remote{
			CACertFilePath: caPath,
			CertFilePath:   clientCertPath,
			KeyFilePath:    clientKeyPath,
			plainHTTP:      plainHTTPNotSpecified,
		},
		Common{},
	}

	uri, err := url.ParseRequestURI(ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	repo, err := opts.NewRepository(uri.Host+"/"+testRepo, opts.Common, logrus.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err = repo.Tags(context.Background(), "", func(got []string) error {
		want := []string{"tag"}
		if len(got) != len(testTagList.Tags) || !reflect.DeepEqual(got, want) {
			return fmt.Errorf("expect: %v, got: %v", testTagList.Tags, got)
		}
		return nil
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemote_NewRepository_Retry(t *testing.T) {
	caPath := filepath.Join(t.TempDir(), "oras-test.pem")
	if err := os.WriteFile(caPath, localhostServerCert, 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	retries, count := 3, 0
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		count++
		if count < retries {
			http.Error(w, "error", http.StatusTooManyRequests)
			return
		}
		err := json.NewEncoder(w).Encode(testTagList)
		if err != nil {
			http.Error(w, "error encoding", http.StatusBadRequest)
		}
	}))
	ts.TLS = loadTestingTLSConfig()
	ts.StartTLS()
	defer ts.Close()
	opts := struct {
		Remote
		Common
	}{
		Remote{
			CACertFilePath: caPath,
			plainHTTP:      plainHTTPNotSpecified,
		},
		Common{},
	}

	uri, err := url.ParseRequestURI(ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	repo, err := opts.NewRepository(uri.Host+"/"+testRepo, opts.Common, logrus.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err = repo.Tags(context.Background(), "", func(got []string) error {
		want := []string{"tag"}
		if len(got) != len(testTagList.Tags) || !reflect.DeepEqual(got, want) {
			return fmt.Errorf("expect: %v, got: %v", testTagList.Tags, got)
		}
		return nil
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != retries {
		t.Errorf("expected %d retries, got %d", retries, count)
	}
}

func TestRemote_default_localhost(t *testing.T) {
	opts := Remote{plainHTTP: plainHTTPNotSpecified}
	got := opts.isPlainHTTP("localhost")
	if got != true {
		t.Fatalf("tls should be disabled when domain is localhost")
	}

	got = opts.isPlainHTTP("localhost:9090")
	if got != true {
		t.Fatalf("tls should be disabled when domain is localhost")
	}
}

func TestRemote_isPlainHTTP_localhost(t *testing.T) {
	opts := Remote{plainHTTP: plainHTTPEnabled}
	isplainHTTP := opts.isPlainHTTP("localhost")
	if isplainHTTP != true {
		t.Fatalf("tls should be disabled when domain is localhost and --plain-http is used")
	}

	isplainHTTP = opts.isPlainHTTP("localhost:9090")
	if isplainHTTP != true {
		t.Fatalf("tls should be disabled when domain is localhost and --plain-http is used")
	}
}

func TestRemote_isHTTPS_localhost(t *testing.T) {
	opts := Remote{plainHTTP: HTTPSEnabled}
	got := opts.isPlainHTTP("localhost")
	if got != false {
		t.Fatalf("tls should be enabled when domain is localhost and --plain-http=false is used")
	}

	got = opts.isPlainHTTP("localhost:9090")
	if got != false {
		t.Fatalf("tls should be enabled when domain is localhost and --plain-http=false is used")
	}
}

func TestRemote_parseResolve_err(t *testing.T) {
	tests := []struct {
		name string
		opts *Remote
	}{
		{
			name: "invalid flag",
			opts: &Remote{resolveFlag: []string{"this-shouldn't_work"}},
		},
		{
			name: "no host",
			opts: &Remote{resolveFlag: []string{":port:address"}},
		},
		{
			name: "no address",
			opts: &Remote{resolveFlag: []string{"host:port:"}},
		},
		{
			name: "invalid address",
			opts: &Remote{resolveFlag: []string{"host:port:invalid-ip"}},
		},
		{
			name: "no port",
			opts: &Remote{resolveFlag: []string{"host::address"}},
		},
		{
			name: "invalid source port",
			opts: &Remote{resolveFlag: []string{"host:port:address"}},
		},
		{
			name: "invalid destination port",
			opts: &Remote{resolveFlag: []string{"host:443:address:port"}},
		},
		{
			name: "no source port",
			opts: &Remote{resolveFlag: []string{"host::address"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.opts.parseResolve(nil); err == nil {
				t.Errorf("Expecting error in Remote.parseResolve()")
			}
		})
	}
}

func TestRemote_parseResolve(t *testing.T) {
	tests := []struct {
		name string
		opts *Remote
	}{
		{
			name: "fromHost:fromPort:toIp",
			opts: &Remote{resolveFlag: []string{"host:443:0.0.0.0"}},
		},
		{
			name: "fromHost:fromPort:toIp:toPort",
			opts: &Remote{resolveFlag: []string{"host:443:0.0.0.0:5000"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.opts.parseResolve(nil); err != nil {
				t.Errorf("Remote.parseResolve() error = %v", err)
			}
		})
	}
}

func TestRemote_parseCustomHeaders(t *testing.T) {
	tests := []struct {
		name        string
		headerFlags []string
		want        http.Header
		wantErr     bool
	}{
		{
			name:        "no custom header is provided",
			headerFlags: []string{},
			want:        nil,
			wantErr:     false,
		},
		{
			name:        "one name-value pair",
			headerFlags: []string{"key:value"},
			want:        map[string][]string{"key": {"value"}},
			wantErr:     false,
		},
		{
			name:        "multiple name-value pairs",
			headerFlags: []string{"key:value", "k:v"},
			want:        map[string][]string{"key": {"value"}, "k": {"v"}},
			wantErr:     false,
		},
		{
			name:        "multiple name-value pairs with commas",
			headerFlags: []string{"key:value,value2,value3", "k:v,v2,v3"},
			want:        map[string][]string{"key": {"value,value2,value3"}, "k": {"v,v2,v3"}},
			wantErr:     false,
		},
		{
			name:        "empty string is a valid value",
			headerFlags: []string{"k:", "key:value,value2,value3"},
			want:        map[string][]string{"k": {""}, "key": {"value,value2,value3"}},
			wantErr:     false,
		},
		{
			name:        "multiple colons are allowed",
			headerFlags: []string{"k::::v,v2,v3", "key:value,value2,value3"},
			want:        map[string][]string{"k": {":::v,v2,v3"}, "key": {"value,value2,value3"}},
			wantErr:     false,
		},
		{
			name:        "name with spaces",
			headerFlags: []string{"bar   :b"},
			want:        map[string][]string{"bar   ": {"b"}},
			wantErr:     false,
		},
		{
			name:        "value with spaces",
			headerFlags: []string{"foo:   a"},
			want:        map[string][]string{"foo": {"   a"}},
			wantErr:     false,
		},
		{
			name:        "repeated pairs",
			headerFlags: []string{"key:value", "key:value"},
			want:        map[string][]string{"key": {"value", "value"}},
			wantErr:     false,
		},
		{
			name:        "repeated name with different values",
			headerFlags: []string{"key:value", "key:value2"},
			want:        map[string][]string{"key": {"value", "value2"}},
			wantErr:     false,
		},
		{
			name:        "one valid header and one invalid header(no pair)",
			headerFlags: []string{"key:value,value2,value3", "vk"},
			want:        nil,
			wantErr:     true,
		},
		{
			name:        "one valid header and one invalid header(empty name)",
			headerFlags: []string{":v", "key:value,value2,value3"},
			want:        nil,
			wantErr:     true,
		},
		{
			name:        "pure-space name is invalid",
			headerFlags: []string{" :  foo "},
			want:        nil,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &Remote{
				headerFlags: tt.headerFlags,
			}
			if err := opts.parseCustomHeaders(); (err != nil) != tt.wantErr {
				t.Errorf("Remote.parseCustomHeaders() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.want, opts.headers) {
				t.Errorf("Remote.parseCustomHeaders() = %v, want %v", opts.headers, tt.want)
			}
		})
	}
}

func TestSameRegistryOrigin(t *testing.T) {
	tests := []struct {
		name           string
		scheme         string
		authority      string
		registryScheme string
		registry       string
		want           bool
	}{
		{"same authority", "https", "registry.example:5443", "https", "registry.example:5443", true},
		{"case insensitive host", "HTTPS", "REGISTRY.EXAMPLE:5443", "https", "registry.example:5443", true},
		{"implicit HTTPS port", "https", "registry.example", "https", "registry.example:443", true},
		{"explicit HTTPS port", "https", "registry.example:443", "https", "registry.example", true},
		{"empty explicit HTTPS port", "https", "registry.example:", "https", "registry.example", true},
		{"plain HTTP registry", "http", "registry.example", "http", "registry.example:80", true},
		{"different port", "https", "registry.example:5443", "https", "registry.example:443", false},
		{"different host", "https", "redirect.example:443", "https", "registry.example:443", false},
		{"different scheme", "http", "registry.example:443", "https", "registry.example:443", false},
		{"same IPv4 literal", "https", "127.0.0.1:5443", "https", "127.0.0.1:5443", true},
		{"compressed and expanded IPv6", "https", "[::1]:5443", "https", "[0:0:0:0:0:0:0:1]:5443", true},
		{"uppercase IPv6 hex digits", "https", "[fe80::ABCD]:5443", "https", "[fe80::abcd]:5443", true},
		{"IPv6 implicit HTTPS port", "https", "[::1]", "https", "[::1]:443", true},
		{"IPv6 explicit HTTPS port", "https", "[::1]:443", "https", "[::1]", true},
		{"IPv6 different port", "https", "[::1]:5443", "https", "[::1]:443", false},
		{"different IPv6 address", "https", "[::2]:443", "https", "[::1]:443", false},
		{"IPv6 zone identifier case differs", "https", "[fe80::1%eth0]:443", "https", "[fe80::1%ETH0]:443", false},
		{"same IPv6 zone identifier", "https", "[fe80::1%eth0]:443", "https", "[fe80::1%eth0]:443", true},
		{"IPv6 zone identifier only on registry", "https", "[fe80::1]:443", "https", "[fe80::1%eth0]:443", false},
		{"IP literal against DNS name", "https", "[::1]:443", "https", "registry.example:443", false},
		{"DNS name against IP literal", "https", "registry.example:443", "https", "[::1]:443", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sameRegistryOrigin(tt.scheme, tt.authority, tt.registryScheme, tt.registry); got != tt.want {
				t.Fatalf("sameRegistryOrigin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSameCredentialHost(t *testing.T) {
	tests := []struct {
		name             string
		resourceRegistry string
		registry         string
		plainHTTP        bool
		want             bool
	}{
		{"implicit HTTPS port on registry", "example.com:443", "example.com", false, true},
		{"implicit HTTPS port on resource", "example.com", "example.com:443", false, true},
		{"implicit HTTP port on registry", "reg:80", "reg", true, true},
		{"implicit HTTP port on resource", "reg", "reg:80", true, true},
		{"non-default port", "example.com:5000", "example.com", false, false},
		{"HTTPS default port under HTTP", "reg:443", "reg", true, false},
		{"case-insensitive host", "EXAMPLE.COM:443", "example.com", false, true},
		{"Docker Hub alias", "registry-1.docker.io:443", "docker.io", false, true},
		{"Docker Hub alias in reverse", "docker.io", "registry-1.docker.io:443", false, true},
		{"IPv6 implicit HTTPS port", "[::1]:443", "::1", false, true},
		{"different host", "example.com:443", "other.example", false, false},
		{"empty registry", "example.com:443", "", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sameCredentialHost(tt.resourceRegistry, tt.registry, tt.plainHTTP); got != tt.want {
				t.Fatalf("sameCredentialHost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemote_authClient_StaticCredentialDefaultPort(t *testing.T) {
	const (
		cliUser = "cli-user"
		cliPass = "cli-pass"
	)
	configPath := writeTestAuthConfig(t, map[string]credentials.Credential{})
	wantCred := credentials.Credential{Username: cliUser, Password: cliPass}

	tests := []struct {
		name             string
		registry         string
		resourceRegistry string
		plainHTTP        bool
		want             bool
	}{
		{"HTTPS default port", "example.com", "example.com:443", false, true},
		{"HTTPS default port on both sides", "example.com", "example.com", false, true},
		{"HTTP default port", "reg", "reg:80", true, true},
		{"non-default port stays in the store", "example.com", "example.com:5000", false, false},
		{"HTTPS default port under HTTP stays in the store", "reg", "reg:443", true, false},
		{"Docker Hub alias", "docker.io", "registry-1.docker.io:443", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Remote{
				Username: cliUser,
				Secret:   cliPass,
				Configs:  []string{configPath},
			}
			client, err := opts.authClient(tt.registry, tt.plainHTTP, false)
			if err != nil {
				t.Fatalf("authClient() error = %v", err)
			}
			got, err := client.CredentialFunc(t.Context(), properties.Resource{Registry: tt.resourceRegistry})
			if err != nil {
				t.Fatalf("CredentialFunc() error = %v", err)
			}
			if tt.want {
				if got != wantCred {
					t.Fatalf("CredentialFunc() = %+v, want %+v", got, wantCred)
				}
				return
			}
			if got != credentials.EmptyCredential {
				t.Fatalf("CredentialFunc() = %+v, want empty credential", got)
			}
		})
	}
}

func TestRemoteNewRegistryScopesClientCertificateToEffectiveHost(t *testing.T) {
	tempDir := t.TempDir()
	clientCertPath := filepath.Join(tempDir, "oras-test-client.pem")
	if err := os.WriteFile(clientCertPath, localhostClientCert, 0600); err != nil {
		t.Fatal(err)
	}
	clientKeyPath := filepath.Join(tempDir, "oras-test-client.key")
	if err := os.WriteFile(clientKeyPath, testingKey(localhostClientKey), 0600); err != nil {
		t.Fatal(err)
	}
	opts := Remote{
		CertFilePath: clientCertPath,
		KeyFilePath:  clientKeyPath,
		plainHTTP:    plainHTTPNotSpecified,
	}
	reg, err := opts.NewRegistry("docker.io", Common{}, logrus.New())
	if err != nil {
		t.Fatal(err)
	}
	client, ok := reg.Client.(*auth.Client)
	if !ok {
		t.Fatalf("registry client has type %T, want *auth.Client", reg.Client)
	}
	transport, ok := client.Client.Transport.(*registryCredentialTransport)
	if !ok {
		t.Fatalf("registry transport has type %T, want *registryCredentialTransport", client.Client.Transport)
	}
	if got, want := transport.registry, "registry-1.docker.io"; got != want {
		t.Fatalf("client certificate registry = %q, want %q", got, want)
	}
}

func TestRemote_authClientScopesClientCertificateToRegistry(t *testing.T) {
	sinkCertificate := make(chan bool, 1)
	sink := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sinkCertificate <- len(r.TLS.PeerCertificates) > 0
		w.WriteHeader(http.StatusNoContent)
	}))
	sink.TLS = loadTestingTLSConfig()
	sink.StartTLS()
	defer sink.Close()

	sourceCertificate := make(chan bool, 1)
	source := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sourceCertificate <- len(r.TLS.PeerCertificates) > 0
		http.Redirect(w, r, sink.URL, http.StatusFound)
	}))
	source.TLS = loadTestingTLSConfig()
	source.StartTLS()
	defer source.Close()

	tempDir := t.TempDir()
	caPath := filepath.Join(tempDir, "oras-test.pem")
	if err := os.WriteFile(caPath, localhostServerCert, 0600); err != nil {
		t.Fatal(err)
	}
	clientCertPath := filepath.Join(tempDir, "oras-test-client.pem")
	if err := os.WriteFile(clientCertPath, localhostClientCert, 0600); err != nil {
		t.Fatal(err)
	}
	clientKeyPath := filepath.Join(tempDir, "oras-test-client.key")
	if err := os.WriteFile(clientKeyPath, testingKey(localhostClientKey), 0600); err != nil {
		t.Fatal(err)
	}

	sourceURL, err := url.Parse(source.URL)
	if err != nil {
		t.Fatal(err)
	}
	opts := Remote{
		CACertFilePath: caPath,
		CertFilePath:   clientCertPath,
		KeyFilePath:    clientKeyPath,
	}
	client, err := opts.authClient(sourceURL.Host, false, false)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, source.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}

	if !<-sourceCertificate {
		t.Fatal("client certificate was not sent to the configured registry")
	}
	if <-sinkCertificate {
		t.Fatal("client certificate was sent to a different redirect origin")
	}
}

func TestRemote_authClientDoesNotSendClientCertificateToHTTPSProxy(t *testing.T) {
	proxyCertificate := make(chan bool, 2)
	var registryAddress string
	proxy := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyCertificate <- len(r.TLS.PeerCertificates) > 0
		if r.Method != http.MethodConnect {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Host != registryAddress {
			http.Error(w, "unexpected CONNECT target", http.StatusForbidden)
			return
		}
		upstream, err := (&net.Dialer{}).DialContext(r.Context(), "tcp", registryAddress)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			_ = upstream.Close()
			http.Error(w, "proxy does not support hijacking", http.StatusInternalServerError)
			return
		}
		downstream, rw, err := hijacker.Hijack()
		if err != nil {
			_ = upstream.Close()
			return
		}
		if _, err := rw.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
			_ = downstream.Close()
			_ = upstream.Close()
			return
		}
		if err := rw.Flush(); err != nil {
			_ = downstream.Close()
			_ = upstream.Close()
			return
		}
		go func() {
			defer downstream.Close()
			defer upstream.Close()
			_, _ = io.Copy(downstream, upstream)
		}()
		go func() {
			defer downstream.Close()
			defer upstream.Close()
			_, _ = io.Copy(upstream, downstream)
		}()
	}))
	proxy.TLS = loadTestingTLSConfig()
	proxy.TLS.ClientAuth = tls.RequestClientCert
	proxy.StartTLS()
	defer proxy.Close()

	registryCertificate := make(chan bool, 1)
	registry := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		registryCertificate <- len(r.TLS.PeerCertificates) > 0
		w.WriteHeader(http.StatusNoContent)
	}))
	registry.TLS = loadTestingTLSConfig()
	registry.TLS.ClientAuth = tls.RequestClientCert
	registry.StartTLS()
	defer registry.Close()
	registryURL, err := url.Parse(registry.URL)
	if err != nil {
		t.Fatal(err)
	}
	registryAddress = registryURL.Host

	tempDir := t.TempDir()
	caPath := filepath.Join(tempDir, "oras-test.pem")
	if err := os.WriteFile(caPath, localhostServerCert, 0600); err != nil {
		t.Fatal(err)
	}
	clientCertPath := filepath.Join(tempDir, "oras-test-client.pem")
	if err := os.WriteFile(clientCertPath, localhostClientCert, 0600); err != nil {
		t.Fatal(err)
	}
	clientKeyPath := filepath.Join(tempDir, "oras-test-client.key")
	if err := os.WriteFile(clientKeyPath, testingKey(localhostClientKey), 0600); err != nil {
		t.Fatal(err)
	}
	opts := Remote{
		CACertFilePath: caPath,
		CertFilePath:   clientCertPath,
		KeyFilePath:    clientKeyPath,
	}
	proxyURL, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		registry   string
		plainHTTP  bool
		requestURL string
	}{
		{"HTTPS registry", registryURL.Host, false, registry.URL},
		{"plain HTTP registry", "registry.example", true, "http://registry.example/v2/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := opts.authClient(tt.registry, tt.plainHTTP, false)
			if err != nil {
				t.Fatal(err)
			}
			scopedTransport, ok := client.Client.Transport.(*registryCredentialTransport)
			if !ok {
				t.Fatalf("registry transport has type %T, want *registryCredentialTransport", client.Client.Transport)
			}
			retryTransport, ok := scopedTransport.registryTransport.(*retry.Transport)
			if !ok {
				t.Fatalf("registry route has type %T, want *retry.Transport", scopedTransport.registryTransport)
			}
			baseTransport, ok := retryTransport.Base.(*http.Transport)
			if !ok {
				t.Fatalf("registry retry base has type %T, want *http.Transport", retryTransport.Base)
			}
			baseTransport.Proxy = http.ProxyURL(proxyURL)
			defer baseTransport.CloseIdleConnections()

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, tt.requestURL, nil)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			if err := resp.Body.Close(); err != nil {
				t.Fatal(err)
			}
			if presented := <-proxyCertificate; presented {
				t.Fatal("registry client certificate was sent to the HTTPS proxy")
			}
			if !tt.plainHTTP && !<-registryCertificate {
				t.Fatal("registry did not receive its client certificate through the HTTPS proxy")
			}
		})
	}
}

func TestRemote_authClientPreservesTLSHandshakeTimeout(t *testing.T) {
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			accepted <- conn
		}
	}()
	defer func() {
		_ = listener.Close()
		select {
		case conn := <-accepted:
			_ = conn.Close()
		default:
		}
	}()

	tempDir := t.TempDir()
	clientCertPath := filepath.Join(tempDir, "oras-test-client.pem")
	if err := os.WriteFile(clientCertPath, localhostClientCert, 0600); err != nil {
		t.Fatal(err)
	}
	clientKeyPath := filepath.Join(tempDir, "oras-test-client.key")
	if err := os.WriteFile(clientKeyPath, testingKey(localhostClientKey), 0600); err != nil {
		t.Fatal(err)
	}
	opts := Remote{
		CertFilePath: clientCertPath,
		KeyFilePath:  clientKeyPath,
	}
	client, err := opts.authClient(listener.Addr().String(), false, false)
	if err != nil {
		t.Fatal(err)
	}
	scopedTransport := client.Client.Transport.(*registryCredentialTransport)
	retryTransport := scopedTransport.registryTransport.(*retry.Transport)
	baseTransport := retryTransport.Base.(*http.Transport)
	baseTransport.TLSHandshakeTimeout = 20 * time.Millisecond
	defer baseTransport.CloseIdleConnections()

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"https://"+listener.Addr().String()+"/v2/",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	resp, err := baseTransport.RoundTrip(req)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err == nil {
		t.Fatal("expected TLS handshake timeout")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("TLS handshake timeout took %v", elapsed)
	}
}

func TestRemote_authClientScopesCustomHeadersToRegistry(t *testing.T) {
	const secret = "private-api-key"

	sinkHeader := make(chan string, 1)
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sinkHeader <- r.Header.Get("X-API-Key")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer sink.Close()

	sourceHeader := make(chan string, 1)
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sourceHeader <- r.Header.Get("X-API-Key")
		http.Redirect(w, r, sink.URL, http.StatusFound)
	}))
	defer source.Close()

	sourceURL, err := url.Parse(source.URL)
	if err != nil {
		t.Fatal(err)
	}
	opts := Remote{headerFlags: []string{"X-API-Key: " + secret}}
	if err := opts.parseCustomHeaders(); err != nil {
		t.Fatal(err)
	}
	client, err := opts.authClient(sourceURL.Host, true, false)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, source.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}

	if got := <-sourceHeader; got != secret {
		t.Fatalf("configured registry received X-API-Key %q, want %q", got, secret)
	}
	if got := <-sinkHeader; got != "" {
		t.Fatalf("different redirect origin received X-API-Key %q", got)
	}
}

func TestRemote_authClientDoesNotShareHeaderState(t *testing.T) {
	received := make(chan http.Header, 1)
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.Header.Clone()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer sink.Close()

	opts := Remote{headerFlags: []string{"X-API-Key: private-api-key"}}
	if err := opts.parseCustomHeaders(); err != nil {
		t.Fatal(err)
	}
	if _, err := opts.authClient("registry.example", true, false); err != nil {
		t.Fatal(err)
	}
	client, err := opts.authClient("registry.example", true, false)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, sink.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}

	header := <-received
	if got := header.Get("X-API-Key"); got != "" {
		t.Fatalf("different origin received X-API-Key %q", got)
	}
	if got := header.Get("User-Agent"); !strings.HasPrefix(got, "oras/") {
		t.Fatalf("different origin received User-Agent %q, want oras/ prefix", got)
	}
}

func TestRemote_NewRepository_referrersAPI(t *testing.T) {
	tests := []struct {
		name  string
		state referrersState
		want  properties.ReferrersAPI
	}{
		{"unknown", ReferrersStateUnknown, properties.ReferrersAPIUnknown},
		{"supported", ReferrersStateSupported, properties.ReferrersAPISupported},
		{"unsupported", ReferrersStateUnsupported, properties.ReferrersAPIUnsupported},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Remote{
				DistributionSpec: DistributionSpec{ReferrersAPI: tt.state},
				plainHTTP:        plainHTTPNotSpecified,
			}
			repo, err := opts.NewRepository("localhost:5000/test", Common{}, logrus.New())
			if err != nil {
				t.Fatalf("Remote.NewRepository() error = %v", err)
			}
			if got := repo.ReferrersCapability(); got != tt.want {
				t.Errorf("Repository.ReferrersCapability() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemote_authClient_dockerHubCredential(t *testing.T) {
	// docker.io credentials are saved under the legacy index key, but requests
	// go to registry-1.docker.io.
	configPath := filepath.Join(t.TempDir(), "config.json")
	encoded := base64.StdEncoding.EncodeToString([]byte("mocked-user:mocked-pass"))
	config := fmt.Sprintf(`{"auths":{"https://index.docker.io/v1/":{"auth":%q}}}`, encoded)
	if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	opts := Remote{Configs: []string{configPath}}
	client, err := opts.authClient("registry-1.docker.io", false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := client.CredentialFunc(context.Background(), properties.Resource{Registry: "registry-1.docker.io"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Username != "mocked-user" || got.Password != "mocked-pass" {
		t.Fatalf("expect credential from index.docker.io key, got: %v", got)
	}
}

func TestRemote_NewRegistry_scheme(t *testing.T) {
	opts := Remote{}
	if _, err := opts.NewRegistry("http://localhost:5000", Common{}, logrus.New()); err == nil {
		t.Fatal("expect NewRegistry to reject a URL scheme")
	}
	if _, err := opts.NewRepository("http://localhost:5000/repo", Common{}, logrus.New()); err == nil {
		t.Fatal("expect NewRepository to reject a URL scheme")
	}
}

func TestRemote_warningHandler(t *testing.T) {
	logger, hook := test.NewNullLogger()
	opts := Remote{plainHTTP: plainHTTPNotSpecified}
	newWarning := func(text string) remote.Warning {
		return remote.Warning{WarningValue: remote.WarningValue{Code: 299, Agent: "-", Text: text}}
	}

	// handlers for the same registry share dedup state across clients
	repo, err := opts.NewRepository("localhost:5000/repo", Common{}, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	reg, err := opts.NewRegistry("localhost:5000", Common{}, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	repo.Registry.HandleWarning(newWarning("foo"))
	repo.Registry.HandleWarning(newWarning("foo"))
	reg.HandleWarning(newWarning("foo"))
	reg.HandleWarning(newWarning("bar"))

	// a different registry logs separately
	other, err := opts.NewRegistry("localhost:6000", Common{}, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	other.HandleWarning(newWarning("foo"))
	other.HandleWarning(newWarning("foo"))

	want := []struct {
		registry string
		text     string
	}{
		{"localhost:5000", "foo"},
		{"localhost:5000", "bar"},
		{"localhost:6000", "foo"},
	}
	entries := hook.AllEntries()
	if len(entries) != len(want) {
		t.Fatalf("got %d log entries, want %d: %v", len(entries), len(want), entries)
	}
	for i, w := range want {
		if entries[i].Level != logrus.WarnLevel {
			t.Errorf("entry %d level = %v, want %v", i, entries[i].Level, logrus.WarnLevel)
		}
		if entries[i].Message != w.text {
			t.Errorf("entry %d message = %q, want %q", i, entries[i].Message, w.text)
		}
		if got := entries[i].Data["registry"]; got != w.registry {
			t.Errorf("entry %d registry = %v, want %v", i, got, w.registry)
		}
	}
}

func TestRemote_ConfigPath_independentOfAuthClient(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	opts := Remote{Configs: []string{configPath}}
	got, err := opts.ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error = %v", err)
	}
	if got != configPath {
		t.Errorf("ConfigPath() = %q, want %q", got, configPath)
	}

	decorated := opts.DecorateCredentialError(auth.ErrBasicCredentialNotFound)
	wantQuotedPath := fmt.Sprintf("at %q", configPath)
	if !strings.Contains(decorated.Recommendation, wantQuotedPath) {
		t.Errorf("DecorateCredentialError() recommendation = %q, want substring %q", decorated.Recommendation, wantQuotedPath)
	}

	secondConfig := filepath.Join(dir, "config2.json")
	if err := os.WriteFile(secondConfig, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	multiOpts := Remote{Configs: []string{configPath, secondConfig}}
	if _, err := multiOpts.ConfigPath(); err == nil {
		t.Error("ConfigPath() with multiple config files = nil, want error")
	}

	invalidConfig := filepath.Join(dir, "invalid.json")
	if err := os.WriteFile(invalidConfig, []byte(`{invalid`), 0o600); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	invalidOpts := Remote{Configs: []string{invalidConfig}}
	if _, err := invalidOpts.ConfigPath(); err == nil {
		t.Error("ConfigPath() with invalid config JSON = nil, want error")
	}
}

func TestRemote_authClient_StaticCredentialScopedToRegistryHost(t *testing.T) {
	t.Run("username and password override store on registry host case-insensitively", func(t *testing.T) {
		const (
			cliUser = "cli-user"
			cliPass = "cli-pass"
		)
		wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(cliUser+":"+cliPass))
		var gotAuth string
		registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != wantAuth {
				w.Header().Set("Www-Authenticate", `Basic realm="registry"`)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			gotAuth = r.Header.Get("Authorization")
			w.WriteHeader(http.StatusOK)
		}))
		defer registry.Close()

		registryURL, err := url.Parse(registry.URL)
		if err != nil {
			t.Fatal(err)
		}
		configPath := writeTestAuthConfig(t, map[string]credentials.Credential{
			registryURL.Host: {Username: "store-user", Password: "store-pass"},
		})
		opts := Remote{
			Username:  cliUser,
			Secret:    cliPass,
			Configs:   []string{configPath},
			plainHTTP: plainHTTPEnabled,
		}
		client, err := opts.authClient(registryURL.Host, true, false)
		if err != nil {
			t.Fatalf("authClient() error = %v", err)
		}

		// Case-insensitive host lookup returns static CLI credential.
		gotCred, err := client.CredentialFunc(t.Context(), properties.Resource{Registry: strings.ToUpper(registryURL.Host)})
		if err != nil {
			t.Fatalf("CredentialFunc() error = %v", err)
		}
		wantCred := credentials.Credential{Username: cliUser, Password: cliPass}
		if gotCred != wantCred {
			t.Fatalf("CredentialFunc() = %+v, want %+v", gotCred, wantCred)
		}

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, registry.URL+"/v2/", nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("client.Do() error = %v", err)
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if gotAuth != wantAuth {
			t.Fatalf("registry received Authorization %q, want %q", gotAuth, wantAuth)
		}
	})

	t.Run("identity token is used on registry host", func(t *testing.T) {
		const (
			idToken     = "cli-identity-token"
			accessToken = "issued-access-token"
		)
		var (
			gotRefreshToken string
			gotBearerAuth   string
		)
		var registryURL string
		registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/oauth2/token":
				if err := r.ParseForm(); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				gotRefreshToken = r.Form.Get("refresh_token")
				if gotRefreshToken != idToken {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprintf(w, `{"access_token":%q}`, accessToken)
			case "/v2/":
				if r.Header.Get("Authorization") != "Bearer "+accessToken {
					w.Header().Set("Www-Authenticate", fmt.Sprintf(`Bearer realm="%s/oauth2/token",service="registry"`, registryURL))
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				gotBearerAuth = r.Header.Get("Authorization")
				w.WriteHeader(http.StatusOK)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer registry.Close()
		registryURL = registry.URL

		parsedURL, err := url.Parse(registry.URL)
		if err != nil {
			t.Fatal(err)
		}
		configPath := writeTestAuthConfig(t, map[string]credentials.Credential{
			parsedURL.Host: {Username: "store-user", Password: "store-pass"},
		})
		opts := Remote{
			Secret:    idToken,
			Configs:   []string{configPath},
			plainHTTP: plainHTTPEnabled,
		}
		client, err := opts.authClient(parsedURL.Host, true, false)
		if err != nil {
			t.Fatalf("authClient() error = %v", err)
		}

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, registry.URL+"/v2/", nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("client.Do() error = %v", err)
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if gotRefreshToken != idToken {
			t.Fatalf("token endpoint received refresh_token %q, want %q", gotRefreshToken, idToken)
		}
		if gotBearerAuth != "Bearer "+accessToken {
			t.Fatalf("registry received Authorization %q, want %q", gotBearerAuth, "Bearer "+accessToken)
		}
	})
}

func TestRemote_authClient_CrossHostUploadLocationUsesStore(t *testing.T) {
	const (
		cliUser     = "cli-user"
		cliPass     = "cli-pass"
		uploadUser  = "upload-user"
		uploadPass  = "upload-pass"
		sessionPath = "/v2/test/blobs/uploads/session"
	)
	cliAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(cliUser+":"+cliPass))
	uploadStoreAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(uploadUser+":"+uploadPass))
	blob := []byte("cross-host-upload-payload")
	blobDesc := ocispec.Descriptor{
		MediaType: "application/octet-stream",
		Digest:    digest.FromBytes(blob),
		Size:      int64(len(blob)),
	}

	runUploadTest := func(t *testing.T, withStoreCred bool) {
		t.Helper()
		var (
			mu         sync.Mutex
			uploadAuth []string
			gotBlob    []byte
		)
		uploadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			mu.Lock()
			uploadAuth = append(uploadAuth, authHeader)
			mu.Unlock()
			if !withStoreCred || authHeader != uploadStoreAuth {
				w.Header().Set("Www-Authenticate", `Basic realm="uploads"`)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			var err error
			gotBlob, err = io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("ReadAll() error = %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Docker-Content-Digest", blobDesc.Digest.String())
			w.WriteHeader(http.StatusCreated)
		}))
		defer uploadServer.Close()

		var registryAuth []string
		registryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			mu.Lock()
			registryAuth = append(registryAuth, authHeader)
			mu.Unlock()
			if authHeader != cliAuth {
				w.Header().Set("Www-Authenticate", `Basic realm="registry"`)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if r.Method == http.MethodPost && r.URL.Path == "/v2/test/blobs/uploads/" {
				w.Header().Set("Location", uploadServer.URL+sessionPath)
				w.WriteHeader(http.StatusAccepted)
				return
			}
			w.WriteHeader(http.StatusForbidden)
		}))
		defer registryServer.Close()

		registryURL, err := url.Parse(registryServer.URL)
		if err != nil {
			t.Fatal(err)
		}
		uploadURL, err := url.Parse(uploadServer.URL)
		if err != nil {
			t.Fatal(err)
		}

		storeAuths := map[string]credentials.Credential{}
		if withStoreCred {
			storeAuths[uploadURL.Host] = credentials.Credential{Username: uploadUser, Password: uploadPass}
		}
		opts := Remote{
			Username:  cliUser,
			Secret:    cliPass,
			Configs:   []string{writeTestAuthConfig(t, storeAuths)},
			plainHTTP: plainHTTPEnabled,
		}
		repo, err := opts.NewRepository(registryURL.Host+"/test", Common{}, logrus.New())
		if err != nil {
			t.Fatalf("NewRepository() error = %v", err)
		}

		pushErr := repo.Blobs().Push(t.Context(), blobDesc, bytes.NewReader(blob))
		if withStoreCred {
			if pushErr != nil {
				t.Fatalf("Blobs().Push() error = %v", pushErr)
			}
			if !bytes.Equal(gotBlob, blob) {
				t.Fatalf("uploaded blob = %q, want %q", gotBlob, blob)
			}
			if len(uploadAuth) < 2 || uploadAuth[len(uploadAuth)-1] != uploadStoreAuth {
				t.Fatalf("upload host authorizations = %v, want final auth %q", uploadAuth, uploadStoreAuth)
			}
		} else {
			if !errors.Is(pushErr, auth.ErrBasicCredentialNotFound) {
				t.Fatalf("Blobs().Push() error = %v, want %v", pushErr, auth.ErrBasicCredentialNotFound)
			}
		}

		for i, got := range uploadAuth {
			if got == cliAuth {
				t.Fatalf("upload host request %d leaked static CLI credential", i)
			}
		}
		if len(registryAuth) == 0 || registryAuth[len(registryAuth)-1] != cliAuth {
			t.Fatalf("registry authorizations = %v, want final auth %q", registryAuth, cliAuth)
		}
	}

	t.Run("uses store credential for upload host", func(t *testing.T) {
		runUploadTest(t, true)
	})
	t.Run("does not leak static credential when store has no entry", func(t *testing.T) {
		runUploadTest(t, false)
	})
}

func TestRemote_authClient_RedirectToOtherHostUsesStore(t *testing.T) {
	const (
		cliUser      = "cli-user"
		cliPass      = "cli-pass"
		redirectUser = "redirect-user"
		redirectPass = "redirect-pass"
	)
	cliAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(cliUser+":"+cliPass))
	redirectStoreAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(redirectUser+":"+redirectPass))

	var (
		mu           sync.Mutex
		redirectAuth []string
	)
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		mu.Lock()
		redirectAuth = append(redirectAuth, authHeader)
		mu.Unlock()
		if authHeader != redirectStoreAuth {
			w.Header().Set("Www-Authenticate", `Basic realm="redirect-target"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer redirectServer.Close()

	sourceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != cliAuth {
			w.Header().Set("Www-Authenticate", `Basic realm="source"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		http.Redirect(w, r, redirectServer.URL+"/blob", http.StatusFound)
	}))
	defer sourceServer.Close()

	sourceURL, err := url.Parse(sourceServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	redirectURL, err := url.Parse(redirectServer.URL)
	if err != nil {
		t.Fatal(err)
	}

	configPath := writeTestAuthConfig(t, map[string]credentials.Credential{
		redirectURL.Host: {Username: redirectUser, Password: redirectPass},
	})
	opts := Remote{
		Username:  cliUser,
		Secret:    cliPass,
		Configs:   []string{configPath},
		plainHTTP: plainHTTPEnabled,
	}

	t.Run("automatic redirect never sends static credential", func(t *testing.T) {
		mu.Lock()
		redirectAuth = nil
		mu.Unlock()

		client, err := opts.authClient(sourceURL.Host, true, false)
		if err != nil {
			t.Fatalf("authClient() error = %v", err)
		}
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, sourceServer.URL+"/v2/test/blobs/sha256:deadbeef", nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("client.Do() error = %v", err)
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
		}
		mu.Lock()
		gotAuths := append([]string(nil), redirectAuth...)
		mu.Unlock()
		for i, got := range gotAuths {
			if got != "" {
				t.Fatalf("redirect target request %d received unexpected Authorization %q", i, got)
			}
		}
	})

	t.Run("redirect target host resolves credential from store", func(t *testing.T) {
		mu.Lock()
		redirectAuth = nil
		mu.Unlock()

		client, err := opts.authClient(sourceURL.Host, true, false)
		if err != nil {
			t.Fatalf("authClient() error = %v", err)
		}
		gotCred, err := client.CredentialFunc(t.Context(), properties.Resource{Registry: redirectURL.Host})
		if err != nil {
			t.Fatalf("CredentialFunc() error = %v", err)
		}
		wantCred := credentials.Credential{Username: redirectUser, Password: redirectPass}
		if gotCred != wantCred {
			t.Fatalf("CredentialFunc() = %+v, want %+v", gotCred, wantCred)
		}

		// Simulate a client following the redirect target URL with the same auth.Client.
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, redirectServer.URL+"/blob", nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("client.Do() error = %v", err)
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		mu.Lock()
		gotAuths := append([]string(nil), redirectAuth...)
		mu.Unlock()
		if len(gotAuths) < 2 || gotAuths[len(gotAuths)-1] != redirectStoreAuth {
			t.Fatalf("redirect target authorizations = %v, want final auth %q", gotAuths, redirectStoreAuth)
		}
		for i, got := range gotAuths {
			if got == cliAuth {
				t.Fatalf("redirect target request %d leaked static CLI credential", i)
			}
		}
	})
}

func TestRemote_authClient_DockerIOAliasScoping(t *testing.T) {
	cliCred := credentials.Credential{Username: "cli-user", Password: "cli-pass"}
	hubStoreCred := credentials.Credential{Username: "hub-store-user", Password: "hub-store-pass"}
	otherStoreCred := credentials.Credential{Username: "other-store-user", Password: "other-store-pass"}

	configPath := writeTestAuthConfig(t, map[string]credentials.Credential{
		"https://index.docker.io/v1/": hubStoreCred,
		"other.example":               otherStoreCred,
	})
	opts := Remote{
		Username:  cliCred.Username,
		Secret:    cliCred.Password,
		Configs:   []string{configPath},
		plainHTTP: plainHTTPNotSpecified,
	}

	for _, regName := range []string{"docker.io", "DOCKER.IO", "registry-1.docker.io"} {
		t.Run("NewRegistry/"+regName, func(t *testing.T) {
			reg, err := opts.NewRegistry(regName, Common{}, logrus.New())
			if err != nil {
				t.Fatalf("NewRegistry() error = %v", err)
			}
			client, ok := reg.Client.(*auth.Client)
			if !ok {
				t.Fatalf("client has type %T, want *auth.Client", reg.Client)
			}
			for _, lookup := range []string{"registry-1.docker.io", "REGISTRY-1.DOCKER.IO", "docker.io", "DOCKER.IO"} {
				got, err := client.CredentialFunc(t.Context(), properties.Resource{Registry: lookup})
				if err != nil {
					t.Fatalf("CredentialFunc(%q) error = %v", lookup, err)
				}
				if got != cliCred {
					t.Fatalf("CredentialFunc(%q) = %+v, want %+v", lookup, got, cliCred)
				}
			}
			gotOther, err := client.CredentialFunc(t.Context(), properties.Resource{Registry: "other.example"})
			if err != nil {
				t.Fatalf("CredentialFunc(other.example) error = %v", err)
			}
			if gotOther != otherStoreCred {
				t.Fatalf("CredentialFunc(other.example) = %+v, want %+v", gotOther, otherStoreCred)
			}
		})
	}

	t.Run("direct authClient with docker.io", func(t *testing.T) {
		client, err := opts.authClient("docker.io", false, false)
		if err != nil {
			t.Fatalf("authClient() error = %v", err)
		}
		got, err := client.CredentialFunc(t.Context(), properties.Resource{Registry: "registry-1.docker.io"})
		if err != nil {
			t.Fatalf("CredentialFunc() error = %v", err)
		}
		if got != cliCred {
			t.Fatalf("CredentialFunc() = %+v, want %+v", got, cliCred)
		}
	})

	t.Run("other registry falls back to docker hub store entry", func(t *testing.T) {
		client, err := opts.authClient("other.example", false, false)
		if err != nil {
			t.Fatalf("authClient() error = %v", err)
		}
		for _, lookup := range []string{"registry-1.docker.io", "docker.io"} {
			got, err := client.CredentialFunc(t.Context(), properties.Resource{Registry: lookup})
			if err != nil {
				t.Fatalf("CredentialFunc(%q) error = %v", lookup, err)
			}
			if got != hubStoreCred {
				t.Fatalf("CredentialFunc(%q) = %+v, want %+v", lookup, got, hubStoreCred)
			}
		}
	})
}

func TestRemote_authClient_DeferredStoreError(t *testing.T) {
	invalidConfig := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(invalidConfig, []byte(`{invalid`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Run("empty CLI credential fails fast on invalid config", func(t *testing.T) {
		opts := Remote{Configs: []string{invalidConfig}}
		if _, err := opts.authClient("registry.example", false, false); err == nil {
			t.Fatal("authClient() error = nil, want invalid config error")
		}
	})

	t.Run("static CLI credential defers store error to non-registry host", func(t *testing.T) {
		want := credentials.Credential{Username: "cli-user", Password: "cli-pass"}
		opts := Remote{
			Username: want.Username,
			Secret:   want.Password,
			Configs:  []string{invalidConfig},
		}
		client, err := opts.authClient("registry.example", false, false)
		if err != nil {
			t.Fatalf("authClient() error = %v, want nil when static credential is set", err)
		}
		got, err := client.CredentialFunc(t.Context(), properties.Resource{Registry: "registry.example"})
		if err != nil {
			t.Fatalf("CredentialFunc(registry.example) error = %v", err)
		}
		if got != want {
			t.Fatalf("CredentialFunc(registry.example) = %+v, want %+v", got, want)
		}
		if _, err := client.CredentialFunc(t.Context(), properties.Resource{Registry: "other.example"}); err == nil {
			t.Fatal("CredentialFunc(other.example) error = nil, want deferred store error")
		}
	})
}
