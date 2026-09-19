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

func TestNewPull_files(t *testing.T) {
	file := File{
		Path: "/out/file.txt",
		Descriptor: FromDescriptor("localhost:5000/test", ocispec.Descriptor{
			MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
			Digest:    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			Size:      42,
		}),
	}
	tests := []struct {
		name  string
		files []File
		want  string
	}{
		{"nil files", nil, "[]"},
		{"empty files", []File{}, "[]"},
		{"one file", []File{file}, `[{"path":"/out/file.txt"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := json.Marshal(NewPull("localhost:5000/test@sha256:abc", tt.files))
			if err != nil {
				t.Fatalf("json.Marshal(NewPull()) error = %v", err)
			}
			var got map[string]json.RawMessage
			if err := json.Unmarshal(content, &got); err != nil {
				t.Fatalf("NewPull() marshalled to invalid JSON %s: %v", content, err)
			}
			raw := string(got["files"])
			if tt.want == "[]" {
				if raw != "[]" {
					t.Errorf("NewPull() files = %s, want []", raw)
				}
			} else if len(raw) < len(tt.want) || raw[:len(tt.want)] != tt.want {
				t.Errorf("NewPull() files = %s, want it to start with %s", raw, tt.want)
			}
		})
	}
}
