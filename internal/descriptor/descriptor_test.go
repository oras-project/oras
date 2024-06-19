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

package descriptor_test

import (
	"reflect"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/internal/descriptor"
)

var (
	artifactDesc = ocispec.Descriptor{
		MediaType: "application/vnd.cncf.oras.artifact.manifest.v1+json",
		Digest:    "sha256:772fbebcda7e6937de01295bae28360afd463c2d5f1f7aca59a3ef267608bc66",
		Size:      568,
	}

	imageDesc = ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    "sha256:2e0e0fe1fb3edbcdddad941c90d2b51e25a6bcd593e82545441a216de7bfa834",
		Size:      474,
	}

	titledDesc = ocispec.Descriptor{
		MediaType:   "application/vnd.oci.image.manifest.v1+json",
		Digest:      "sha256:2e0e0fe1fb3edbcdddad941c90d2b51e25a6bcd593e82545441a216de7bfa834",
		Size:        474,
		Annotations: map[string]string{"org.opencontainers.image.title": "shaboozey"},
	}
)

func TestDescriptor_IsImageManifest(t *testing.T) {
	got := descriptor.IsImageManifest(imageDesc)
	if !reflect.DeepEqual(got, true) {
		t.Fatalf("IsImageManifest() got %v, want %v", got, true)
	}

	got = descriptor.IsImageManifest(artifactDesc)
	if !reflect.DeepEqual(got, false) {
		t.Fatalf("IsImageManifest() got %v, want %v", got, false)
	}
}

func TestDescriptor_ShortDigest(t *testing.T) {
	expected := "2e0e0fe1fb3e"
	got := descriptor.ShortDigest(titledDesc)
	if expected != got {
		t.Fatalf("GetTitleOrMediaType() got %v, want %v", got, expected)
	}
}

func TestDescriptor_GetTitleOrMediaType(t *testing.T) {
	expected := "application/vnd.oci.image.manifest.v1+json"
	name, isTitle := descriptor.GetTitleOrMediaType(imageDesc)
	if expected != name {
		t.Fatalf("GetTitleOrMediaType() got %v, want %v", name, expected)
	}
	if false != isTitle {
		t.Fatalf("GetTitleOrMediaType() got %v, want %v", isTitle, false)
	}

	expected = "shaboozey"
	name, isTitle = descriptor.GetTitleOrMediaType(titledDesc)
	if expected != name {
		t.Fatalf("GetTitleOrMediaType() got %v, want %v", name, expected)
	}
	if true != isTitle {
		t.Fatalf("GetTitleOrMediaType() got %v, want %v", isTitle, false)
	}
}

func TestDescriptor_GenerateContentKey(t *testing.T) {
	expected := "sha256:2e0e0fe1fb3edbcdddad941c90d2b51e25a6bcd593e82545441a216de7bfa834shaboozey"
	got := descriptor.GenerateContentKey(titledDesc)
	if expected != got {
		t.Fatalf("GetTitleOrMediaType() got %v, want %v", got, expected)
	}
}
