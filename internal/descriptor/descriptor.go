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

package descriptor

import (
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/internal/docker"
)

// IsImageManifest checks whether a manifest is an image manifest.
func IsImageManifest(desc ocispec.Descriptor) bool {
	return desc.MediaType == docker.MediaTypeManifest || desc.MediaType == ocispec.MediaTypeImageManifest
}

// ShortDigest converts the digest of the descriptor to a short form for displaying.
func ShortDigest(desc ocispec.Descriptor) (digestString string) {
	digestString = desc.Digest.String()
	if err := desc.Digest.Validate(); err == nil {
		if algo := desc.Digest.Algorithm(); algo == digest.SHA256 {
			digestString = desc.Digest.Encoded()[:12]
		}
	}
	return digestString
}

// GetTitleOrMediaType gets a descriptor name using either title or media type.
func GetTitleOrMediaType(desc ocispec.Descriptor) (name string, isTitle bool) {
	name, ok := desc.Annotations[ocispec.AnnotationTitle]
	if !ok {
		return desc.MediaType, false
	}
	return name, true
}

// GenerateContentKey generates a unique key for each content descriptor using
// digest and name.
func GenerateContentKey(desc ocispec.Descriptor) string {
	return desc.Digest.String() + desc.Annotations[ocispec.AnnotationTitle]
}
