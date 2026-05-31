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
package template

import (
	"bytes"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestManifestFetchHandler_OnFetched(t *testing.T) {
	manifestContent := []byte(`{
		"schemaVersion": 2,
		"mediaType": "application/vnd.oci.image.manifest.v1+json",
		"config": {
			"mediaType": "application/vnd.oci.image.config.v1+json",
			"digest": "sha256:abc123",
			"size": 100
		},
		"annotations": {
			"org.opencontainers.image.revision": "abc123"
		}
	}`)

	desc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    digest.FromString("test"),
		Size:      100,
	}

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "top level mediaType works",
			template: "{{ .mediaType }}",
			want:     "application/vnd.oci.image.manifest.v1+json",
		},
		{
			name:     "config.mediaType resolves correctly",
			template: "{{ .config.mediaType }}",
			want:     "application/vnd.oci.image.config.v1+json",
		},
		{
			name:     "manifest annotations resolve correctly",
			template: `{{ index .annotations "org.opencontainers.image.revision" }}`,
			want:     "abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			handler := NewManifestFetchHandler(buf, tt.template)
			err := handler.(*manifestFetchHandler).OnFetched("localhost:5000/test", desc, manifestContent)
			if err != nil {
				t.Fatalf("OnFetched() error = %v", err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("OnFetched() output = %q, want %q", got, tt.want)
			}
		})
	}
}
