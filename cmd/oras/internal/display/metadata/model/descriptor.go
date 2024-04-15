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
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// DigestReference is a reference to an artifact with digest.
type DigestReference struct {
	Ref string `json:"ref"`
}

// NewDigestReference creates a new digest reference.
func NewDigestReference(name string, digest string) DigestReference {
	return DigestReference{
		Ref: name + "@" + digest,
	}
}

// Descriptor is a descriptor with digest reference.
// We cannot use ocispec.Descriptor here since the first letter of the json
// annotation key is not uppercase.
type Descriptor struct {
	DigestReference
	ocispec.Descriptor
}

// FromDescriptor converts a OCI descriptor to a descriptor with digest reference.
func FromDescriptor(name string, desc ocispec.Descriptor) Descriptor {
	ret := Descriptor{
		DigestReference: NewDigestReference(name, desc.Digest.String()),
		Descriptor:      desc,
	}
	return ret
}
