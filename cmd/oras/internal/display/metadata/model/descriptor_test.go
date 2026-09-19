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

const descTestDigest = "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

func TestNewDigestReference(t *testing.T) {
	got := NewDigestReference("localhost:5000/test", descTestDigest)
	if want := "localhost:5000/test@" + descTestDigest; got.Reference != want {
		t.Errorf("NewDigestReference() = %q, want %q", got.Reference, want)
	}
}

func TestFromDescriptor(t *testing.T) {
	desc := ocispec.Descriptor{
		MediaType:    "application/vnd.oci.image.manifest.v1+json",
		Digest:       descTestDigest,
		Size:         100,
		ArtifactType: "application/vnd.example.sbom",
		Annotations:  map[string]string{ocispec.AnnotationTitle: "file.txt"},
	}
	got := FromDescriptor("localhost:5000/test", desc)

	if want := "localhost:5000/test@" + descTestDigest; got.Reference != want {
		t.Errorf("FromDescriptor() reference = %q, want %q", got.Reference, want)
	}
	if got.MediaType != desc.MediaType || got.Digest != desc.Digest || got.Size != desc.Size {
		t.Errorf("FromDescriptor() core fields = %v, want them copied from %v", got.Descriptor, desc)
	}
	if got.ArtifactType != desc.ArtifactType {
		t.Errorf("FromDescriptor() artifactType = %q, want %q", got.ArtifactType, desc.ArtifactType)
	}
	if got.Annotations[ocispec.AnnotationTitle] != "file.txt" {
		t.Errorf("FromDescriptor() annotations = %v, want the title preserved", got.Annotations)
	}
}

// FromDescriptor copies a subset of the OCI descriptor, so fields outside that
// subset are dropped rather than carried into the output.
func TestFromDescriptor_dropsUnsupportedFields(t *testing.T) {
	desc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    descTestDigest,
		Size:      100,
		URLs:      []string{"https://example.test/blob"},
		Data:      []byte("inline"),
		Platform:  &ocispec.Platform{OS: "linux", Architecture: "amd64"},
	}
	got := FromDescriptor("localhost:5000/test", desc)

	if got.URLs != nil {
		t.Errorf("FromDescriptor() urls = %v, want nil", got.URLs)
	}
	if got.Data != nil {
		t.Errorf("FromDescriptor() data = %v, want nil", got.Data)
	}
	if got.Platform != nil {
		t.Errorf("FromDescriptor() platform = %v, want nil", got.Platform)
	}
}

// The embedded OCI fields must be inlined alongside the reference rather than
// nested, since templates address them as .reference and .mediaType.
func TestDescriptor_jsonShape(t *testing.T) {
	content, err := json.Marshal(FromDescriptor("localhost:5000/test", ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    descTestDigest,
		Size:      100,
	}))
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var got map[string]json.RawMessage
	if err := json.Unmarshal(content, &got); err != nil {
		t.Fatalf("marshalled to invalid JSON %s: %v", content, err)
	}
	for _, key := range []string{"reference", "mediaType", "digest", "size"} {
		if _, ok := got[key]; !ok {
			t.Errorf("Descriptor JSON is missing key %q; got %s", key, content)
		}
	}
}
