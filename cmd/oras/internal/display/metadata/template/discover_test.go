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
	"strings"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const referrerDigest = "sha256:9f6ad1a2d9ec1d0e4b5b1f9f6c9c9a4c6e9f1d4b7a2c5e8f1a4d7b0c3e6f9a2d"

var testReferrer = ocispec.Descriptor{
	MediaType:    "application/vnd.oci.image.manifest.v1+json",
	ArtifactType: "application/vnd.example.sbom",
	Digest:       referrerDigest,
	Size:         42,
}

func TestNewDiscoverHandler(t *testing.T) {
	if got := NewDiscoverHandler(&bytes.Buffer{}, testDescriptor, "localhost:5000/test", "{{.reference}}"); got == nil {
		t.Fatal("NewDiscoverHandler() returned nil")
	}
}

func TestDiscoverHandler_OnDiscovered(t *testing.T) {
	handler := NewDiscoverHandler(&bytes.Buffer{}, testDescriptor, "localhost:5000/test", "{{.reference}}")

	// the root is a known subject, so its referrer is accepted
	if err := handler.OnDiscovered(testReferrer, testDescriptor); err != nil {
		t.Fatalf("OnDiscovered() error = %v, want nil", err)
	}

	// a referrer of the first referrer is accepted too, since discovering a
	// node registers it as a possible subject
	nested := testReferrer
	nested.Digest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	if err := handler.OnDiscovered(nested, testReferrer); err != nil {
		t.Errorf("OnDiscovered() error = %v, want nil for a referrer of a known referrer", err)
	}
}

func TestDiscoverHandler_OnDiscovered_unknownSubject(t *testing.T) {
	handler := NewDiscoverHandler(&bytes.Buffer{}, testDescriptor, "localhost:5000/test", "{{.reference}}")

	unknown := testDescriptor
	unknown.Digest = "sha256:1111111111111111111111111111111111111111111111111111111111111111"

	err := handler.OnDiscovered(testReferrer, unknown)
	if err == nil {
		t.Fatal("OnDiscovered() error = nil, want an error for an unknown subject")
	}
	if !strings.Contains(err.Error(), "unexpected subject descriptor") {
		t.Errorf("OnDiscovered() error = %q, want it to mention an unexpected subject descriptor", err.Error())
	}
}

func TestDiscoverHandler_Render(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewDiscoverHandler(buf, testDescriptor, "localhost:5000/test", "{{.reference}}|{{len .referrers}}")

	if err := handler.OnDiscovered(testReferrer, testDescriptor); err != nil {
		t.Fatalf("OnDiscovered() error = %v, want nil", err)
	}
	if err := handler.Render(); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	want := "localhost:5000/test@" + testDigest + "|1"
	if got := buf.String(); got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestDiscoverHandler_Render_invalidTemplate(t *testing.T) {
	handler := NewDiscoverHandler(&bytes.Buffer{}, testDescriptor, "localhost:5000/test", "{{.reference")
	if err := handler.Render(); err == nil {
		t.Error("Render() error = nil, want an error for an unparsable template")
	}
}
