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

package model

import (
	"encoding/json"
	"testing"
)

// The list is initialised to an empty slice, so an empty listing marshals as []
// rather than null.
func TestNewRepositories(t *testing.T) {
	r := NewRepositories("localhost:5000")
	if r.Registry != "localhost:5000" {
		t.Errorf("NewRepositories() registry = %q, want %q", r.Registry, "localhost:5000")
	}

	content, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal(content, &got); err != nil {
		t.Fatalf("marshalled to invalid JSON %s: %v", content, err)
	}
	if string(got["repositories"]) != "[]" {
		t.Errorf("NewRepositories() repositories = %s, want []", got["repositories"])
	}
}

func TestRepositories_AddRepository(t *testing.T) {
	r := NewRepositories("localhost:5000")
	for _, repo := range []string{"team/alpha", "team/beta"} {
		r.AddRepository(repo)
	}

	if len(r.Repositories) != 2 {
		t.Fatalf("AddRepository() repositories = %v, want 2 entries", r.Repositories)
	}
	// Insertion order is preserved here, unlike Tagged.Tags, which sorts.
	if r.Repositories[0] != "team/alpha" || r.Repositories[1] != "team/beta" {
		t.Errorf("AddRepository() repositories = %v, want them in insertion order", r.Repositories)
	}
}
