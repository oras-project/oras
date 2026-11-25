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

func TestRepoTagsHandler_Render(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewRepoTagsHandler(buf)
	for _, tag := range []string{"v1.0.0", "latest", "v2.0.0"} {
		if err := handler.OnTagListed(tag); err != nil {
			t.Fatalf("OnTagListed(%q) error = %v, want nil", tag, err)
		}
	}
	if err := handler.Render(); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	var result struct {
		Tags []string `json:"tags"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Render() produced invalid JSON: %v", err)
	}
	want := []string{"v1.0.0", "latest", "v2.0.0"}
	if !slices.Equal(result.Tags, want) {
		t.Errorf("Render() tags = %v, want %v", result.Tags, want)
	}
}

func TestRepoTagsHandler_Render_empty(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewRepoTagsHandler(buf)
	if err := handler.Render(); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	var result map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Render() produced invalid JSON: %v", err)
	}
	if got := string(result["tags"]); got != "[]" {
		t.Errorf("Render() tags = %s, want []", got)
	}
}
