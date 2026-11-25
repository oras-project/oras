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

var pushTestDescriptor = ocispec.Descriptor{
	MediaType: "application/vnd.oci.image.manifest.v1+json",
	Digest:    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	Size:      100,
}

func TestNewPush_referenceAsTags(t *testing.T) {
	tests := []struct {
		name string
		tags []string
		want string
	}{
		{"nil tags", nil, `[]`},
		{"empty tags", []string{}, `[]`},
		{"one tag", []string{"v1"}, `["localhost:5000/test:v1"]`},
		{"two tags", []string{"latest", "v1"}, `["localhost:5000/test:latest","localhost:5000/test:v1"]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := json.Marshal(NewPush(pushTestDescriptor, "localhost:5000/test", tt.tags))
			if err != nil {
				t.Fatalf("json.Marshal(NewPush()) error = %v", err)
			}
			var got map[string]json.RawMessage
			if err := json.Unmarshal(content, &got); err != nil {
				t.Fatalf("NewPush() marshalled to invalid JSON %s: %v", content, err)
			}
			if string(got["referenceAsTags"]) != tt.want {
				t.Errorf("NewPush() referenceAsTags = %s, want %s", got["referenceAsTags"], tt.want)
			}
		})
	}
}
