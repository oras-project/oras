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

package text

import (
	"bytes"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/output"
)

func TestManifestIndexCreateHandler_OnTagged(t *testing.T) {
	mockPrinter := output.NewPrinter(bytes.NewBuffer(nil), bytes.NewBuffer(nil), false)
	handler := NewManifestIndexCreateHandler(mockPrinter)
	
	desc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageIndex,
		Digest:    "sha256:test1234",
		Size:      1024,
	}
	
	err := handler.OnTagged(desc, "v1.0.0")
	if err != nil {
		t.Errorf("OnTagged() error = %v, want nil", err)
	}
}

func TestManifestIndexCreateHandler_OnIndexCreated(t *testing.T) {
	mockPrinter := output.NewPrinter(bytes.NewBuffer(nil), bytes.NewBuffer(nil), false)
	handler := NewManifestIndexCreateHandler(mockPrinter)
	
	desc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageIndex,
		Digest:    "sha256:abc123",
		Size:      2048,
	}
	
	concreteHandler := handler.(*ManifestIndexCreateHandler)
	concreteHandler.OnIndexCreated(desc)
	
	if concreteHandler.root.Digest != desc.Digest {
		t.Errorf("OnIndexCreated() root digest = %v, want %v", concreteHandler.root.Digest, desc.Digest)
	}
}

func TestManifestIndexCreateHandler_Render(t *testing.T) {
	mockPrinter := output.NewPrinter(bytes.NewBuffer(nil), bytes.NewBuffer(nil), false)
	handler := NewManifestIndexCreateHandler(mockPrinter)
	
	desc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageIndex,
		Digest:    "sha256:render123",
		Size:      512,
	}
	
	concreteHandler := handler.(*ManifestIndexCreateHandler)
	concreteHandler.OnIndexCreated(desc)
	
	err := concreteHandler.Render()
	if err != nil {
		t.Errorf("Render() error = %v, want nil", err)
	}
}

func TestNewManifestIndexCreateHandler(t *testing.T) {
	mockPrinter := output.NewPrinter(bytes.NewBuffer(nil), bytes.NewBuffer(nil), false)
	
	handler := NewManifestIndexCreateHandler(mockPrinter)
	if handler == nil {
		t.Error("NewManifestIndexCreateHandler() returned nil")
	}
	
	concreteHandler, ok := handler.(*ManifestIndexCreateHandler)
	if !ok {
		t.Error("NewManifestIndexCreateHandler() did not return *ManifestIndexCreateHandler")
	}
	
	if concreteHandler.printer != mockPrinter {
		t.Error("NewManifestIndexCreateHandler() printer not set correctly")
	}
}

