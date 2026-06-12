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

func TestNewPushHandler(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewPushHandler(buf)
	if handler == nil {
		t.Fatal("NewPushHandler() returned nil")
	}
}

func TestPushHandler_OnTagged(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewPushHandler(buf).(*PushHandler)

	desc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		Size:      100,
	}

	err := handler.OnTagged(desc, "v1.0.0")
	if err != nil {
		t.Errorf("PushHandler.OnTagged() error = %v, want nil", err)
	}
}

func TestPushHandler_OnCopied(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewPushHandler(buf).(*PushHandler)

	desc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		Size:      100,
	}

	opts := &option.Target{
		RawReference: "localhost:5000/test:v1.0.0",
		Path:         "localhost:5000/test",
		Reference:    "v1.0.0",
	}

	err := handler.OnCopied(opts, desc)
	if err != nil {
		t.Errorf("PushHandler.OnCopied() error = %v, want nil", err)
	}
}

func TestPushHandler_OnCopied_WithDigest(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewPushHandler(buf).(*PushHandler)

	desc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		Size:      100,
	}

	opts := &option.Target{
		RawReference: "localhost:5000/test@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		Path:         "localhost:5000/test",
		Reference:    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	}

	err := handler.OnCopied(opts, desc)
	if err != nil {
		t.Errorf("PushHandler.OnCopied() error = %v, want nil", err)
	}
}

func TestPushHandler_Render(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewPushHandler(buf).(*PushHandler)

	desc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		Size:      100,
	}

	opts := &option.Target{
		RawReference: "localhost:5000/test:v1.0.0",
		Path:         "localhost:5000/test",
		Reference:    "v1.0.0",
	}

	if err := handler.OnCopied(opts, desc); err != nil {
		t.Fatalf("PushHandler.OnCopied() error = %v", err)
	}

	err := handler.Render()
	if err != nil {
		t.Errorf("PushHandler.Render() error = %v, want nil", err)
	}

	// Verify JSON output is valid
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Errorf("PushHandler.Render() produced invalid JSON: %v", err)
	}
}
