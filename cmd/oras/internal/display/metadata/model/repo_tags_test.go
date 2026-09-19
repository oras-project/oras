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
func TestNewTags(t *testing.T) {
	content, err := json.Marshal(NewTags())
	if err != nil {
		t.Fatalf("json.Marshal(NewTags()) error = %v", err)
	}

	var got map[string]json.RawMessage
	if err := json.Unmarshal(content, &got); err != nil {
		t.Fatalf("marshalled to invalid JSON %s: %v", content, err)
	}
	if string(got["tags"]) != "[]" {
		t.Errorf("NewTags() tags = %s, want []", got["tags"])
	}
}

func TestTags_AddTag(t *testing.T) {
	tags := NewTags()
	for _, tag := range []string{"v1.0.0", "latest"} {
		tags.AddTag(tag)
	}

	if len(tags.Tags) != 2 {
		t.Fatalf("AddTag() tags = %v, want 2 entries", tags.Tags)
	}
	if tags.Tags[0] != "v1.0.0" || tags.Tags[1] != "latest" {
		t.Errorf("AddTag() tags = %v, want them in insertion order", tags.Tags)
	}
}
