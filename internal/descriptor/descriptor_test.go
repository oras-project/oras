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

func TestIsImageManifest(t *testing.T) {
	imageDesc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    "sha256:2e0e0fe1fb3edbcdddad941c90d2b51e25a6bcd593e82545441a216de7bfa834",
		Size:      474,
	}

	got := descriptor.IsImageManifest(imageDesc)
	if !reflect.DeepEqual(got, true) {
		t.Fatalf("IsImageManifest() got %v, want %v", got, true)
	}

	artifactDesc := ocispec.Descriptor{
		MediaType: "application/vnd.cncf.oras.artifact.manifest.v1+json",
		Digest:    "sha256:772fbebcda7e6937de01295bae28360afd463c2d5f1f7aca59a3ef267608bc66",
		Size:      568,
	}

	got = descriptor.IsImageManifest(artifactDesc)
	if !reflect.DeepEqual(got, false) {
		t.Fatalf("IsImageManifest() got %v, want %v", got, false)
	}
}
