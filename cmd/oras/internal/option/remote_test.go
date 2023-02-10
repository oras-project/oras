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
	"context"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	nhttp "net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"

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

func TestMain(m *testing.M) {
	// Test server
	ts = httptest.NewTLSServer(nhttp.HandlerFunc(func(w nhttp.ResponseWriter, r *nhttp.Request) {
		p := r.URL.Path
		m := r.Method
		switch {
		case p == "/v2/" && m == "GET":
			w.WriteHeader(nhttp.StatusOK)
		case p == fmt.Sprintf("/v2/%s/tags/list", testRepo) && m == "GET":
			json.NewEncoder(w).Encode(testTagList)
		}
	}))
	defer ts.Close()
	m.Run()
}

func TestRemote_FlagsInit(t *testing.T) {
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
		Password: want.Password,
	}
	client, err := opts.authClient("hostname", false)
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

func TestRemote_authClient_skipTlsVerify(t *testing.T) {
	opts := Remote{
		Insecure: true,
	}
	client, err := opts.authClient("hostname", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req, err := nhttp.NewRequestWithContext(context.Background(), nhttp.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemote_authClient_CARoots(t *testing.T) {
	caPath := filepath.Join(t.TempDir(), "oras-test.pem")
	if err := os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ts.Certificate().Raw}), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(ts.Certificate())

	opts := Remote{
		CACertFilePath: caPath,
	}
	client, err := opts.authClient("hostname", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req, err := nhttp.NewRequestWithContext(context.Background(), nhttp.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
	client, err := opts.authClient(testHost, false)
	if err != nil {
		t.Fatalf("unexpected error when creating auth client: %v", err)
	}
	req, err := nhttp.NewRequestWithContext(context.Background(), nhttp.MethodGet, fmt.Sprintf("https://%s:%s", testHost, URL.Port()), nil)
	if err != nil {
		t.Fatalf("unexpected error when generating request: %v", err)
	}
	_, err = client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error when sending request: %v", err)
	}
}

func TestRemote_NewRegistry(t *testing.T) {
	caPath := filepath.Join(t.TempDir(), "oras-test.pem")
	if err := os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ts.Certificate().Raw}), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(ts.Certificate())

	opts := struct {
		Remote
		Common
	}{
		Remote{
			CACertFilePath: caPath,
		},
		Common{},
	}
	uri, err := url.ParseRequestURI(ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	reg, err := opts.NewRegistry(uri.Host, opts.Common)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err = reg.Ping(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemote_NewRepository(t *testing.T) {
	caPath := filepath.Join(t.TempDir(), "oras-test.pem")
	if err := os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ts.Certificate().Raw}), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(ts.Certificate())
	opts := struct {
		Remote
		Common
	}{
		Remote{
			CACertFilePath: caPath,
		},
		Common{},
	}

	uri, err := url.ParseRequestURI(ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	repo, err := opts.NewRepository(uri.Host+"/"+testRepo, opts.Common)
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

func TestRemote_isPlainHttp_localhost(t *testing.T) {
	opts := Remote{PlainHTTP: false}
	got := opts.isPlainHttp("localhost")
	if got != true {
		t.Fatalf("tls should be disabled when domain is localhost")

	}

	got = opts.isPlainHttp("localhost:9090")
	if got != true {
		t.Fatalf("tls should be disabled when domain is localhost")

	}
}

func TestRemote_parseResolve_err(t *testing.T) {
	tests := []struct {
		name    string
		opts    *Remote
		wantErr bool
	}{
		{
			name:    "invalid flag",
			opts:    &Remote{resolveFlag: []string{"this-shouldn't_work"}},
			wantErr: true,
		},
		{
			name:    "no host",
			opts:    &Remote{resolveFlag: []string{":port:address"}},
			wantErr: true,
		},
		{
			name:    "no address",
			opts:    &Remote{resolveFlag: []string{"host:port:"}},
			wantErr: true,
		},
		{
			name:    "invalid address",
			opts:    &Remote{resolveFlag: []string{"host:port:invalid-ip"}},
			wantErr: true,
		},
		{
			name:    "no port",
			opts:    &Remote{resolveFlag: []string{"host::address"}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.opts.parseResolve(); (err != nil) != tt.wantErr {
				t.Errorf("Remote.parseResolve() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRemote_parseCustomHeaders(t *testing.T) {
	tests := []struct {
		name        string
		headerFlags []string
		want        nhttp.Header
		wantErr     bool
	}{
		{
			name:        "no custom header is provided",
			headerFlags: []string{},
			want:        nil,
			wantErr:     false,
		},
		{
			name:        "valid input, one header, one value",
			headerFlags: []string{"value:key"},
			want:        map[string][]string{"value": {"key"}},
			wantErr:     false,
		},
		{
			name:        "valid input, multiple header, one value",
			headerFlags: []string{"value:key", "v:k"},
			want:        map[string][]string{"value": {"key"}, "v": {"k"}},
			wantErr:     false,
		},
		{
			name:        "valid input, multiple header, multiple values",
			headerFlags: []string{"value:key,key2,key3", "v:k,k2,k3"},
			want:        map[string][]string{"value": {"key,key2,key3"}, "v": {"k,k2,k3"}},
			wantErr:     false,
		},
		{
			name:        "one valid header and one invalid header(no pair)",
			headerFlags: []string{"value:key,key2,key3", "vk"},
			want:        nil,
			wantErr:     true,
		},
		{
			name:        "one valid header and one invalid header(no value)",
			headerFlags: []string{"vk:", "value:key,key2,key3"},
			want:        nil,
			wantErr:     true,
		},
		{
			name:        "empty string is a valid key",
			headerFlags: []string{":k,k2,k3", "value:key,key2,key3"},
			want:        map[string][]string{"": {"k,k2,k3"}, "value": {"key,key2,key3"}},
			wantErr:     false,
		},
		{
			name:        "multiple colons are allowed",
			headerFlags: []string{"::::k,k2,k3", "value:key,key2,key3"},
			want:        map[string][]string{"": {":::k,k2,k3"}, "value": {"key,key2,key3"}},
			wantErr:     false,
		},
		{
			name:        "invalid headers",
			headerFlags: []string{"foo", "bar:"},
			want:        nil,
			wantErr:     true,
		},
		{
			name:        "valid headers with spaces",
			headerFlags: []string{"foo:   a", "bar   :b"},
			want:        map[string][]string{"foo": {"   a"}, "bar   ": {"b"}},
			wantErr:     false,
		},
		{
			name:        "invalid headers with spaces",
			headerFlags: []string{" foo:   ", "bar: "},
			want:        nil,
			wantErr:     true,
		},
		{
			name:        "repeated headers",
			headerFlags: []string{"value:key", "value:key"},
			want:        map[string][]string{"value": {"key", "key"}},
			wantErr:     false,
		},
		{
			name:        "repeated key with different values",
			headerFlags: []string{"value:key", "value:key2"},
			want:        map[string][]string{"value": {"key", "key2"}},
			wantErr:     false,
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
