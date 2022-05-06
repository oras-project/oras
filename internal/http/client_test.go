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
package http_test

import (
	"testing"

	nhttp "net/http"

	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras/internal/http"
)

func Test_NewClient_credential(t *testing.T) {
	wanted := auth.Credential{
		Username: "username",
	}
	opts := http.ClientOptions{
		Credential: wanted,
	}
	client := http.NewClient(opts)
	got, err := client.(*auth.Client).Credential(nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != wanted {
		t.Fatalf("expect: %v, got: %v", wanted, got)
	}
}

func Test_NewClient_tlsConfig(t *testing.T) {
	opts := http.ClientOptions{
		SkipTLSVerify: true,
	}

	wanted := opts.SkipTLSVerify
	client := http.NewClient(opts)
	config := client.(*auth.Client).Client.Transport.(*nhttp.Transport).TLSClientConfig
	got := config.InsecureSkipVerify
	if got != wanted {
		t.Fatalf("expect: %v, got: %v", wanted, got)
	}
}
