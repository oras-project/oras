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
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	nhttp "net/http"
	"net/http/httptest"

	"oras.land/oras-go/v2/registry/remote/auth"
)

var ts *httptest.Server

func TestMain(m *testing.M) {
	// Test server
	ts = httptest.NewTLSServer(nhttp.HandlerFunc(func(w nhttp.ResponseWriter, r *nhttp.Request) {
		p := r.URL.Path
		m := r.Method
		switch {
		case p == "/v2/" && m == "GET":
			w.WriteHeader(nhttp.StatusOK)
		}
	}))
	defer ts.Close()
	m.Run()
}

func Test_authClient_RawCredential(t *testing.T) {
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
	client, err := opts.authClient(false)
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

func Test_authClient_skipTlsVerify(t *testing.T) {
	opts := Remote{
		Insecure: true,
	}
	client, err := opts.authClient(false)
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

func Test_authClient_CARoots(t *testing.T) {
	caPath := filepath.Join(t.TempDir(), "oras-test.pem")
	if err := os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ts.Certificate().Raw}), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(ts.Certificate())

	opts := Remote{
		CACertFilePath: caPath,
	}
	client, err := opts.authClient(false)
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
