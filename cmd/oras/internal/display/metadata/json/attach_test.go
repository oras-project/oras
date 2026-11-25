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
	"oras.land/oras/cmd/oras/internal/option"
)

func TestAttachHandler_Render(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewAttachHandler(buf)

	root := ocispec.Descriptor{
		MediaType:    "application/vnd.oci.image.manifest.v1+json",
		Digest:       testDigest,
		Size:         100,
		ArtifactType: "application/vnd.example.signature",
	}
	subject := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    "sha256:2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae",
		Size:      200,
	}
	handler.OnAttached(&option.Target{Path: "localhost:5000/test"}, root, subject)

	if err := handler.Render(); err != nil {
		t.Fatalf("AttachHandler.Render() error = %v, want nil", err)
	}

	var result struct {
		Reference    string `json:"reference"`
		MediaType    string `json:"mediaType"`
		Digest       string `json:"digest"`
		Size         int64  `json:"size"`
		ArtifactType string `json:"artifactType"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("AttachHandler.Render() produced invalid JSON: %v", err)
	}

	// The output describes the attached root, not the subject.
	if want := "localhost:5000/test@" + testDigest; result.Reference != want {
		t.Errorf("Render() reference = %q, want %q", result.Reference, want)
	}
	if result.Digest != string(root.Digest) {
		t.Errorf("Render() digest = %q, want %q", result.Digest, root.Digest)
	}
	if result.MediaType != root.MediaType {
		t.Errorf("Render() mediaType = %q, want %q", result.MediaType, root.MediaType)
	}
	if result.Size != root.Size {
		t.Errorf("Render() size = %d, want %d", result.Size, root.Size)
	}
	if result.ArtifactType != root.ArtifactType {
		t.Errorf("Render() artifactType = %q, want %q", result.ArtifactType, root.ArtifactType)
	}
}
