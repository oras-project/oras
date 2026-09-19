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

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

var fetchedDesc = ocispec.Descriptor{
	MediaType: "application/vnd.oci.image.manifest.v1+json",
	Digest:    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	Size:      100,
}

func TestNewFetched(t *testing.T) {
	content, err := json.Marshal(NewFetched("localhost:5000/test", fetchedDesc, map[string]any{"schemaVersion": 2}))
	if err != nil {
		t.Fatalf("json.Marshal(NewFetched()) error = %v", err)
	}

	var got map[string]json.RawMessage
	if err := json.Unmarshal(content, &got); err != nil {
		t.Fatalf("NewFetched() marshalled to invalid JSON %s: %v", content, err)
	}
	if want := `{"schemaVersion":2}`; string(got["content"]) != want {
		t.Errorf("NewFetched() content = %s, want %s", got["content"], want)
	}
	if _, ok := got["reference"]; !ok {
		t.Errorf("NewFetched() is missing the reference key; got %s", content)
	}
}

// Nil content is reported as null rather than being an error, which is what
// lets the fetch handlers tolerate content that is not JSON.
func TestNewFetched_nilContent(t *testing.T) {
	content, err := json.Marshal(NewFetched("localhost:5000/test", fetchedDesc, nil))
	if err != nil {
		t.Fatalf("json.Marshal(NewFetched()) error = %v", err)
	}

	var got map[string]json.RawMessage
	if err := json.Unmarshal(content, &got); err != nil {
		t.Fatalf("NewFetched() marshalled to invalid JSON %s: %v", content, err)
	}
	if string(got["content"]) != "null" {
		t.Errorf("NewFetched() content = %s, want null", got["content"])
	}
}
