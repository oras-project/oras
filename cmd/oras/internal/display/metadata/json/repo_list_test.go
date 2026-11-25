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

package json

import (
	"bytes"
	"encoding/json"
	"slices"
	"testing"
)

func TestRepoListHandler_Render(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewRepoListHandler(buf, "localhost:5000")
	for _, repo := range []string{"team/app", "team/app/sub", "library/nginx"} {
		if err := handler.OnRepositoryListed(repo); err != nil {
			t.Fatalf("OnRepositoryListed(%q) error = %v, want nil", repo, err)
		}
	}
	if err := handler.Render(); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	var result struct {
		Registry     string   `json:"registry"`
		Repositories []string `json:"repositories"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Render() produced invalid JSON: %v", err)
	}
	if result.Registry != "localhost:5000" {
		t.Errorf("Render() registry = %q, want %q", result.Registry, "localhost:5000")
	}
	// JSON output lists full repository names, in the order they were listed.
	want := []string{"team/app", "team/app/sub", "library/nginx"}
	if !slices.Equal(result.Repositories, want) {
		t.Errorf("Render() repositories = %v, want %v", result.Repositories, want)
	}
}

func TestRepoListHandler_Render_empty(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewRepoListHandler(buf, "localhost:5000")
	if err := handler.Render(); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	var result map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Render() produced invalid JSON: %v", err)
	}
	if got := string(result["repositories"]); got != "[]" {
		t.Errorf("Render() repositories = %s, want []", got)
	}
}
