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
		Digest:    testDigest,
		Size:      1024,
	}

	if err := handler.OnLayerSkipped(desc); err != nil {
		t.Errorf("PullHandler.OnLayerSkipped() error = %v, want nil", err)
	}
}

func TestPullHandler_OnFilePulled(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewPullHandler(buf, "localhost:5000/test").(*PullHandler)

	desc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
		Digest:    testDigest,
		Size:      1024,
	}

	if err := handler.OnFilePulled("test.txt", "/output", desc, "blobs/sha256/"+testDigest[len("sha256:"):]); err != nil {
		t.Fatalf("PullHandler.OnFilePulled() error = %v, want nil", err)
	}

	files := handler.pulled.Files()
	if len(files) != 1 {
		t.Fatalf("PullHandler.OnFilePulled() did not record file: got %d files, want 1", len(files))
	}
	if files[0].Digest != desc.Digest {
		t.Errorf("recorded file Digest = %q, want %q", files[0].Digest, desc.Digest)
	}
	if files[0].MediaType != desc.MediaType {
		t.Errorf("recorded file MediaType = %q, want %q", files[0].MediaType, desc.MediaType)
	}
}

func TestPullHandler_OnPulled(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewPullHandler(buf, "localhost:5000/test").(*PullHandler)

	desc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    testDigest,
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

	rootDesc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    testDigest,
		Size:      100,
	}
	fileDesc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
		Digest:    testDigest,
		Size:      1024,
	}

	handler.OnPulled(&option.Target{}, rootDesc)
	if err := handler.OnFilePulled("test.txt", "/output", fileDesc, "blobs/sha256/"+testDigest[len("sha256:"):]); err != nil {
		t.Fatalf("PullHandler.OnFilePulled() error = %v", err)
	}

	if err := handler.Render(); err != nil {
		t.Fatalf("PullHandler.Render() error = %v, want nil", err)
	}

	var result struct {
		Reference string `json:"reference"`
		Files     []struct {
			Path      string `json:"path"`
			Reference string `json:"reference"`
			MediaType string `json:"mediaType"`
			Digest    string `json:"digest"`
			Size      int64  `json:"size"`
		} `json:"files"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("PullHandler.Render() produced invalid JSON: %v", err)
	}

	wantReference := "localhost:5000/test@" + testDigest
	if result.Reference != wantReference {
		t.Errorf("Render() reference = %q, want %q", result.Reference, wantReference)
	}
	if len(result.Files) != 1 {
		t.Fatalf("Render() files length = %d, want 1", len(result.Files))
	}
	if result.Files[0].MediaType != fileDesc.MediaType {
		t.Errorf("Render() files[0].mediaType = %q, want %q", result.Files[0].MediaType, fileDesc.MediaType)
	}
	if result.Files[0].Digest != string(fileDesc.Digest) {
		t.Errorf("Render() files[0].digest = %q, want %q", result.Files[0].Digest, fileDesc.Digest)
	}
	if result.Files[0].Size != fileDesc.Size {
		t.Errorf("Render() files[0].size = %d, want %d", result.Files[0].Size, fileDesc.Size)
	}
}
