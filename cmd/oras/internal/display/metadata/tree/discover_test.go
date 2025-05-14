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

package tree

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestDiscoverHandler_OnDiscovered(t *testing.T) {
	path := "localhost:5000/test"
	subjectDesc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    "sha256:9d16f5505246424aed7116cb21216704ba8c919997d0f1f37e154c11d509e1d2",
		Size:      529,
	}
	referrerDesc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    digest.Digest("sha256:e2c6633a79985906f1ed55c592718c73c41e809fb9818de232a635904a74d48d"),
		Size:      660,
		Annotations: map[string]string{
			"org.opencontainers.image.created": "2023-01-18T08:37:42Z",
		},
		ArtifactType: "test/sbom.file",
	}
	coloredRoot := digestColor.Apply(fmt.Sprintf("%s@%s", path, subjectDesc.Digest))
	coloredArtifactType := artifactTypeColor.Apply(referrerDesc.ArtifactType)
	coloredDigest := digestColor.Apply(referrerDesc.Digest.String())
	coloredAnnotations := annotationsColor.Apply("[annotations]")

	t.Run("WithTTY", func(t *testing.T) {
		var buf bytes.Buffer

		// create a temp file to mock TTY
		tmp, err := os.CreateTemp(t.TempDir(), "test-tty")
		if err != nil {
			t.Fatalf("Failed to create temporary file: %v", err)
		}
		defer func() { _ = os.Remove(tmp.Name()) }()
		defer func() { _ = tmp.Close() }()

		h := NewDiscoverHandler(&buf, path, subjectDesc, true, tmp)
		if err := h.OnDiscovered(referrerDesc, subjectDesc); err != nil {
			t.Fatalf("OnDiscovered() error = %v", err)
		}

		// when rendered to a buffer, the node content should include colors
		if err := h.Render(); err != nil {
			t.Fatalf("Failed to render tree: %v", err)
		}
		output := buf.String()

		// verify root
		if !strings.Contains(output, coloredRoot) {
			t.Errorf("expected root output contains %v, got: %v", coloredRoot, output)
		}

		// verify artifact type
		if !strings.Contains(output, coloredArtifactType) {
			t.Errorf("expected artifact type output contains %v, got: %v", coloredArtifactType, output)
		}

		// verify digest
		if !strings.Contains(output, coloredDigest) {
			t.Errorf("expected digest output contains %v, got: %v", coloredDigest, output)
		}

		// verify annotations
		if !strings.Contains(output, coloredAnnotations) {
			t.Errorf("expected annotations output contains %v, got: %v", coloredAnnotations, output)
		}
	})

	t.Run("WithoutTTY", func(t *testing.T) {
		var buf bytes.Buffer

		h := NewDiscoverHandler(&buf, path, subjectDesc, true, nil)
		if err := h.OnDiscovered(referrerDesc, subjectDesc); err != nil {
			t.Fatalf("OnDiscovered() error = %v", err)
		}

		// when rendered to a buffer, the node content should not include colors
		if err := h.Render(); err != nil {
			t.Fatalf("Failed to render tree: %v", err)
		}
		output := buf.String()

		// verify root
		if strings.Contains(output, coloredRoot) {
			t.Errorf("expected root output not contains %v, got: %v", coloredRoot, output)
		}

		// verify artifact type
		if strings.Contains(output, coloredArtifactType) {
			t.Errorf("expected artifact type output not contains %v, got: %v", coloredArtifactType, output)
		}

		// verify digest
		if strings.Contains(output, coloredDigest) {
			t.Errorf("expected digest output not contains %v, got: %v", coloredDigest, output)
		}

		// verify annotations
		if strings.Contains(output, coloredAnnotations) {
			t.Errorf("expected annotations output not contains %v, got: %v", coloredAnnotations, output)
		}
	})
}
