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

	"oras.land/oras/internal/descriptor"
)

func TestIsImageManifest(t *testing.T) {
	mediaType := "application/vnd.oci.image.manifest.v1+json"
	got := descriptor.IsImageManifest(mediaType)
	if !reflect.DeepEqual(got, true) {
		t.Fatalf("IsImageManifest() got %v, want %v", got, true)
	}

	mediaType = "application/vnd.cncf.oras.artifact.manifest.v1+json"
	got = descriptor.IsImageManifest(mediaType)
	if !reflect.DeepEqual(got, false) {
		t.Fatalf("IsImageManifest() got %v, want %v", got, false)
	}
}
