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

package root

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func Test_tagManifest_remoteNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()
	uri, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd := tagCmd()
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{"--plain-http", uri.Host + "/test:v1", "v2"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("tag error = nil, want not found error")
	}
}
