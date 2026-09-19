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

package model

import (
	"encoding/json"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestNewAttach(t *testing.T) {
	desc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		Size:      100,
	}

	content, err := json.Marshal(NewAttach(desc, "localhost:5000/test"))
	if err != nil {
		t.Fatalf("json.Marshal(NewAttach()) error = %v", err)
	}

	var got struct {
		Reference string `json:"reference"`
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Size      int64  `json:"size"`
	}
	if err := json.Unmarshal(content, &got); err != nil {
		t.Fatalf("NewAttach() marshalled to invalid JSON %s: %v", content, err)
	}

	if want := "localhost:5000/test@" + desc.Digest.String(); got.Reference != want {
		t.Errorf("NewAttach() reference = %q, want %q", got.Reference, want)
	}
	if got.MediaType != desc.MediaType {
		t.Errorf("NewAttach() mediaType = %q, want %q", got.MediaType, desc.MediaType)
	}
	if got.Digest != desc.Digest.String() {
		t.Errorf("NewAttach() digest = %q, want %q", got.Digest, desc.Digest)
	}
	if got.Size != desc.Size {
		t.Errorf("NewAttach() size = %d, want %d", got.Size, desc.Size)
	}
}
