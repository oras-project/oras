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
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
)

func TestManifestPushHandler_OnTagged(t *testing.T) {
	mockPrinter := output.NewPrinter(bytes.NewBuffer(nil), bytes.NewBuffer(nil), false)
	mockTarget := &option.Target{
		RawReference: "test-repo:test-tag",
	}
	handler := NewManifestPushHandler(mockPrinter, mockTarget)
	
	desc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Digest:    "sha256:test1234",
		Size:      1024,
	}
	
	err := handler.OnTagged(desc, "v1.0.0")
	if err != nil {
		t.Errorf("OnTagged() error = %v, want nil", err)
	}
}

func TestManifestPushHandler_OnManifestPushed(t *testing.T) {
	mockPrinter := output.NewPrinter(bytes.NewBuffer(nil), bytes.NewBuffer(nil), false)
	mockTarget := &option.Target{
		RawReference: "test-repo:pushed-tag",
	}
	handler := NewManifestPushHandler(mockPrinter, mockTarget)
	
	desc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Digest:    "sha256:pushed123",
		Size:      2048,
	}
	
	concreteHandler := handler.(*ManifestPushHandler)
	err := concreteHandler.OnManifestPushed(desc)
	if err != nil {
		t.Errorf("OnManifestPushed() error = %v, want nil", err)
	}
	
	if concreteHandler.desc.Digest != desc.Digest {
		t.Errorf("OnManifestPushed() desc digest = %v, want %v", concreteHandler.desc.Digest, desc.Digest)
	}
}

func TestManifestPushHandler_Render(t *testing.T) {
	mockPrinter := output.NewPrinter(bytes.NewBuffer(nil), bytes.NewBuffer(nil), false)
	mockTarget := &option.Target{
		RawReference: "test-repo:render-tag",
	}
	handler := NewManifestPushHandler(mockPrinter, mockTarget)
	
	desc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Digest:    "sha256:render123",
		Size:      512,
	}
	
	concreteHandler := handler.(*ManifestPushHandler)
	concreteHandler.OnManifestPushed(desc)
	
	err := concreteHandler.Render()
	if err != nil {
		t.Errorf("Render() error = %v, want nil", err)
	}
}

func TestNewManifestPushHandler(t *testing.T) {
	mockPrinter := output.NewPrinter(bytes.NewBuffer(nil), bytes.NewBuffer(nil), false)
	mockTarget := &option.Target{
		RawReference: "test-repo:test-tag",
	}
	
	handler := NewManifestPushHandler(mockPrinter, mockTarget)
	if handler == nil {
		t.Error("NewManifestPushHandler() returned nil")
	}
	
	concreteHandler, ok := handler.(*ManifestPushHandler)
	if !ok {
		t.Error("NewManifestPushHandler() did not return *ManifestPushHandler")
	}
	
	if concreteHandler.printer != mockPrinter {
		t.Error("NewManifestPushHandler() printer not set correctly")
	}
	
	if concreteHandler.target != mockTarget {
		t.Error("NewManifestPushHandler() target not set correctly")
	}
}
