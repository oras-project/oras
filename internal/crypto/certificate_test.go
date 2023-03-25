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

package crypto

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

var ts *httptest.Server

func TestLoadCertPool(t *testing.T) {
	// Test server
	ts = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()
	var err error
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	client := &http.Client{}
	_, err = client.Do(req)
	if err == nil {
		t.Fatalf("expecting TLS check failure error but didn't get one")
	}

	tp := http.DefaultTransport.(*http.Transport).Clone()
	caPath := filepath.Join(t.TempDir(), "oras-test.pem")
	if err = os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ts.Certificate().Raw}), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tp.TLSClientConfig.RootCAs, err = LoadCertPool(caPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	client = &http.Client{Transport: tp}
	_, err = client.Do(req)
	if err != nil {
		t.Fatalf("failed to trust the self signed pem: %v", err)
	}
}

func TestLoadCertPool_err(t *testing.T) {
	pemPath := filepath.Join(t.TempDir(), "invalid.pem")
	if err := os.WriteFile(pemPath, []byte{}, 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    *x509.CertPool
		wantErr bool
	}{
		{"should fail if the path is doesn't exist", args{"/???"}, nil, true},
		{"should fail if the file is not a pem", args{pemPath}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadCertPool(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadCertPool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LoadCertPool() = %v, want %v", got, tt.want)
			}
		})
	}
}
