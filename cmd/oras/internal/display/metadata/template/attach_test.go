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

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/option"
)

const testDigest = "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

var testDescriptor = ocispec.Descriptor{
	MediaType: "application/vnd.oci.image.manifest.v1+json",
	Digest:    testDigest,
	Size:      100,
}

func TestNewAttachHandler(t *testing.T) {
	if got := NewAttachHandler(&bytes.Buffer{}, "{{.reference}}"); got == nil {
		t.Fatal("NewAttachHandler() returned nil")
	}
}

func TestAttachHandler_Render(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewAttachHandler(buf, "{{.reference}} {{.size}}").(*AttachHandler)

	handler.OnAttached(&option.Target{Path: "localhost:5000/test"}, testDescriptor, ocispec.Descriptor{})
	if handler.path != "localhost:5000/test" {
		t.Errorf("OnAttached() path = %q, want %q", handler.path, "localhost:5000/test")
	}

	if err := handler.Render(); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}
	want := "localhost:5000/test@" + testDigest + " 100"
	if got := buf.String(); got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestAttachHandler_Render_invalidTemplate(t *testing.T) {
	handler := NewAttachHandler(&bytes.Buffer{}, "{{.reference")
	handler.OnAttached(&option.Target{Path: "localhost:5000/test"}, testDescriptor, ocispec.Descriptor{})
	if err := handler.Render(); err == nil {
		t.Error("Render() error = nil, want an error for an unparsable template")
	}
}
