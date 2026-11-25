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
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestManifestFetchHandler_OnFetched(t *testing.T) {
	desc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    testDigest,
		Size:      100,
	}
	tests := []struct {
		name        string
		content     []byte
		wantContent string
	}{
		{
			name:        "JSON manifest is embedded as content",
			content:     []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json"}`),
			wantContent: `{"mediaType":"application/vnd.oci.image.manifest.v1+json","schemaVersion":2}`,
		},
		{
			name:        "non-JSON content is reported as null",
			content:     []byte("not json"),
			wantContent: "null",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			handler := NewManifestFetchHandler(buf)
			if err := handler.OnFetched("localhost:5000/test", desc, tt.content); err != nil {
				t.Fatalf("OnFetched() error = %v, want nil", err)
			}

			var result map[string]json.RawMessage
			if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
				t.Fatalf("OnFetched() produced invalid JSON: %v", err)
			}
			var reference string
			if err := json.Unmarshal(result["reference"], &reference); err != nil {
				t.Fatalf("OnFetched() reference is not a string: %v", err)
			}
			if want := "localhost:5000/test@" + testDigest; reference != want {
				t.Errorf("OnFetched() reference = %q, want %q", reference, want)
			}

			// Compact the content so the comparison ignores the pretty-printing.
			var got bytes.Buffer
			if err := json.Compact(&got, result["content"]); err != nil {
				t.Fatalf("OnFetched() content is not valid JSON: %v", err)
			}
			if got.String() != tt.wantContent {
				t.Errorf("OnFetched() content = %s, want %s", got.String(), tt.wantContent)
			}
		})
	}
}
