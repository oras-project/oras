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
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"oras.land/oras-go/v2/registry/remote/auth"
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

func TestRemote_authClient_RawCredential(t *testing.T) {
	password := make([]byte, 12)
	if _, err := rand.Read(password); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := auth.Credential{
		Username: "mocked^^??oras-@@!#",
		Password: base64.StdEncoding.EncodeToString(password),
	}
	opts := Remote{
		Username: want.Username,
		Secret:   want.Password,
	}
	client, err := opts.authClient("hostname", false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := client.Credential(nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Username != want.Username || got.Password != want.Password {
		t.Fatalf("expect: %v, got: %v", want, got)
	}
}

func TestRemote_authClient_SharedCache(t *testing.T) {
	ClearSharedClient()
	t.Cleanup(ClearSharedClient)

	opts := Remote{
		Username: "test-user",
		Secret:   "test-password",
	}
	first, err := opts.authClient("hostname", false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := GetSharedClient(); got != first {
		t.Fatalf("expected first client to be shared, got %p", got)
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
