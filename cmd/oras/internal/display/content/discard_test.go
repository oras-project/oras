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

package content

import (
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestNewDiscardHandler(t *testing.T) {
	handler := NewDiscardHandler()
	if handler != (DiscardHandler{}) {
		t.Errorf("NewDiscardHandler() returned unexpected handler")
	}
}

func TestDiscardHandler_OnContentFetched(t *testing.T) {
	handler := NewDiscardHandler()
	desc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		Size:      0,
	}
	content := []byte("test content")

	err := handler.OnContentFetched(desc, content)
	if err != nil {
		t.Errorf("DiscardHandler.OnContentFetched() error = %v, want nil", err)
	}
}

func TestDiscardHandler_OnContentCreated(t *testing.T) {
	handler := NewDiscardHandler()
	content := []byte("test content")

	err := handler.OnContentCreated(content)
	if err != nil {
		t.Errorf("DiscardHandler.OnContentCreated() error = %v, want nil", err)
	}
}

func TestDiscardHandler_ImplementsInterfaces(t *testing.T) {
	handler := NewDiscardHandler()

	// Verify DiscardHandler implements ManifestFetchHandler
	var _ ManifestFetchHandler = handler

	// Verify DiscardHandler implements ManifestIndexCreateHandler
	var _ ManifestIndexCreateHandler = handler
}
