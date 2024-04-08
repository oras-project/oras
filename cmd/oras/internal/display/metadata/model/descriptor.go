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
	"context"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"oras.land/oras/cmd/oras/internal/display/status"
	"sync"
)

// DigestReference is a reference to an artifact with digest.
type DigestReference struct {
	Ref string
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

	// ArtifactType is the IANA media type of this artifact.
	ArtifactType string
}

// FromDescriptor converts a OCI descriptor to a descriptor with digest reference.
func FromDescriptor(name string, desc ocispec.Descriptor) Descriptor {
	ret := Descriptor{
		DigestReference: NewDigestReference(name, desc.Digest.String()),
		MediaType:       desc.MediaType,
		Digest:          desc.Digest,
		Size:            desc.Size,
		URLs:            desc.URLs,
		Annotations:     desc.Annotations,
		ArtifactType:    desc.ArtifactType,
	}
	return ret
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

// PrintDescriptor prints descriptor status and name.
func PrintDescriptor(printer *status.Printer, desc ocispec.Descriptor, status string) {
	name, ok := desc.Annotations[ocispec.AnnotationTitle]
	if !ok {
		name = desc.MediaType
		printer.PrintVerbose(status, ShortDigest(desc), name)
		return
	}
	printer.Print(status, ShortDigest(desc), name)
	return
}

// PrintSuccessorStatus prints transfer status of successors.
func PrintSuccessorStatus(ctx context.Context, desc ocispec.Descriptor, fetcher content.Fetcher, committed *sync.Map, prompt string, printer *status.Printer) error {
	successors, err := content.Successors(ctx, fetcher, desc)
	if err != nil {
		return err
	}
	for _, s := range successors {
		name := s.Annotations[ocispec.AnnotationTitle]
		if v, ok := committed.Load(s.Digest.String()); ok && v != name {
			// Reprint status for deduplicated content
			PrintDescriptor(printer, desc, prompt)
		}
	}
	return nil
}
