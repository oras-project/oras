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

package text

import (
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/output"
	"oras.land/oras/internal/contentutil"
)

// ManifestIndexCreateHandler handles text metadata output for index create events.
type ManifestIndexCreateHandler struct {
	printer *output.Printer
}

// NewManifestIndexCreateHandler returns a new handler for index create events.
func NewManifestIndexCreateHandler(printer *output.Printer) metadata.ManifestIndexCreateHandler {
	return &ManifestIndexCreateHandler{
		printer: printer,
	}
}

// OnIndexPacked implements metadata.ManifestIndexCreateHandler.
func (h *ManifestIndexCreateHandler) OnIndexPacked(shortDigest string) error {
	return h.printer.Println("Packed", shortDigest, ocispec.MediaTypeImageIndex)
}

// OnIndexPushed implements metadata.ManifestIndexCreateHandler.
func (h *ManifestIndexCreateHandler) OnIndexPushed(path string) error {
	return h.printer.Println("Pushed", path)
}

// OnTagged implements metadata.TaggedHandler.
func (h *ManifestIndexCreateHandler) OnTagged(_ ocispec.Descriptor, tag string) error {
	return h.printer.Println("Tagged", tag)
}

// OnCompleted implements metadata.ManifestIndexCreateHandler.
func (h *ManifestIndexCreateHandler) OnCompleted(desc ocispec.Descriptor) error {
	return h.printer.Println("Digest:", desc.Digest)
}

type ManifestIndexUpdateHandler struct {
	printer *output.Printer
}

// NewManifestIndexUpdateHandler returns a new handler for index update events.
func NewManifestIndexUpdateHandler(printer *output.Printer) metadata.ManifestIndexUpdateHandler {
	return &ManifestIndexUpdateHandler{
		printer: printer,
	}
}

// OnManifestRemoved implements metadata.ManifestIndexUpdateHandler.
func (miuh ManifestIndexUpdateHandler) OnManifestRemoved(digest digest.Digest) error {
	return miuh.printer.Println("Removed", digest)
}

// OnManifestAdded implements metadata.ManifestIndexUpdateHandler.
func (miuh ManifestIndexUpdateHandler) OnManifestAdded(ref string, digest digest.Digest) error {
	if contentutil.IsDigest(ref) {
		return miuh.printer.Println("Added", ref)
	}
	return miuh.printer.Println("Added", digest, ref)
}

// OnIndexMerged implements metadata.ManifestIndexUpdateHandler.
func (miuh ManifestIndexUpdateHandler) OnIndexMerged(ref string, digest digest.Digest) error {
	if contentutil.IsDigest(ref) {
		return miuh.printer.Println("Merged", ref)
	}
	return miuh.printer.Println("Merged", digest, ref)
}

// OnIndexUpdated implements metadata.ManifestIndexUpdateHandler.
func (miuh ManifestIndexUpdateHandler) OnIndexUpdated(digest digest.Digest) error {
	return miuh.printer.Println("Updated", digest)
}

// OnIndexPushed implements metadata.ManifestIndexUpdateHandler.
func (miuh ManifestIndexUpdateHandler) OnIndexPushed(indexRef string) error {
	return miuh.printer.Println("Pushed", indexRef)
}

// OnTagged implements metadata.TaggedHandler.
func (h *ManifestIndexUpdateHandler) OnTagged(_ ocispec.Descriptor, tag string) error {
	return h.printer.Println("Tagged", tag)
}

// OnCompleted implements metadata.ManifestIndexUpdateHandler.
func (h *ManifestIndexUpdateHandler) OnCompleted(desc ocispec.Descriptor) error {
	return h.printer.Println("Digest:", desc.Digest)
}
