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

func TestNewPullHandler(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewPullHandler(buf, "localhost:5000/test")
	if handler == nil {
		t.Fatal("NewPullHandler() returned nil")
	}
}

func TestPullHandler_OnLayerSkipped(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewPullHandler(buf, "localhost:5000/test").(*PullHandler)

	desc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
		Digest:    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		Size:      1024,
	}

	err := handler.OnLayerSkipped(desc)
	if err != nil {
		t.Errorf("PullHandler.OnLayerSkipped() error = %v, want nil", err)
	}
}

func TestPullHandler_OnFilePulled(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewPullHandler(buf, "localhost:5000/test").(*PullHandler)

	desc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
		Digest:    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		Size:      1024,
	}

	err := handler.OnFilePulled("test.txt", "/output", desc, "blobs/sha256/e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
	if err != nil {
		t.Errorf("PullHandler.OnFilePulled() error = %v, want nil", err)
	}
}

func TestPullHandler_OnPulled(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewPullHandler(buf, "localhost:5000/test").(*PullHandler)

	desc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		Size:      100,
	}

	handler.OnPulled(&option.Target{}, desc)

	if handler.root.Digest != desc.Digest {
		t.Errorf("PullHandler.OnPulled() did not set root descriptor correctly")
	}
}

func TestPullHandler_Render(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewPullHandler(buf, "localhost:5000/test").(*PullHandler)

	desc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		Size:      100,
	}

	handler.OnPulled(&option.Target{}, desc)

	err := handler.Render()
	if err != nil {
		t.Errorf("PullHandler.Render() error = %v, want nil", err)
	}

	// Verify JSON output is valid
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Errorf("PullHandler.Render() produced invalid JSON: %v", err)
	}
}
