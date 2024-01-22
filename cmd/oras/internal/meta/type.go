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

package meta

import (
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// DigestReference is a reference to an artifact with digest.
type DigestReference struct {
	Ref string
}

// ToDigestReference converts a name and digest to a digest reference.
func ToDigestReference(name string, digest string) DigestReference {
	return DigestReference{
		Ref: name + "@" + digest,
	}
}

// Descriptor is a descriptor with digest reference.
type Descriptor struct {
	DigestReference
	// MediaType is the media type of the object this schema refers to.
	MediaType string

	// Digest is the digest of the targeted content.
	Digest digest.Digest

	// Size specifies the size in bytes of the blob.
	Size int64

	// URLs specifies a list of URLs from which this object MAY be downloaded
	URLs []string `json:",omitempty"`

	// Annotations contains arbitrary metadata relating to the targeted content.
	Annotations map[string]string `json:",omitempty"`

	// Data is an embedding of the targeted content. This is encoded as a base64
	// string when marshalled to JSON (automatically, by encoding/json). If
	// present, Data can be used directly to avoid fetching the targeted content.
	Data []byte `json:",omitempty"`

	// Platform describes the platform which the image in the manifest runs on.
	//
	// This should only be used when referring to a manifest.
	Platform *Platform `json:",omitempty"`

	// ArtifactType is the IANA media type of this artifact.
	ArtifactType string
}

// Platform describes the platform which the image in the manifest runs on.
type Platform struct {
	// Architecture field specifies the CPU architecture, for example
	// `amd64` or `ppc64le`.
	Architecture string

	// OS specifies the operating system, for example `linux` or `windows`.
	OS string

	// Variant is an optional field specifying a variant of the CPU, for
	// example `v7` to specify ARMv7 when architecture is `arm`.
	Variant string
}

// ToDescriptor converts a descriptor to a descriptor with digest reference.
func ToDescriptor(name string, desc ocispec.Descriptor) Descriptor {
	ret := Descriptor{
		DigestReference: ToDigestReference(name, desc.Digest.String()),
		MediaType:       desc.MediaType,
		Digest:          desc.Digest,
		Size:            desc.Size,
		URLs:            desc.URLs,
		Annotations:     desc.Annotations,
		Data:            desc.Data,
	}
	return ret
}
