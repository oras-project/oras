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
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/output"
)

// ManifestIndexCreateHandler handles text metadata output for index create events.
type ManifestIndexCreateHandler struct {
	printer *output.Printer
	root    ocispec.Descriptor
}

// NewManifestIndexCreateHandler returns a new handler for index create events.
func NewManifestIndexCreateHandler(printer *output.Printer) metadata.ManifestIndexCreateHandler {
	return &ManifestIndexCreateHandler{
		printer: printer,
	}
}

// OnTagged implements TaggedHandler.
func (h *ManifestIndexCreateHandler) OnTagged(_ ocispec.Descriptor, tag string) error {
	return h.printer.Println("Tagged", tag)
}

// OnIndexCreated implements ManifestIndexCreateHandler.
func (h *ManifestIndexCreateHandler) OnIndexCreated(desc ocispec.Descriptor) {
	h.root = desc
}

// Render implements ManifestIndexCreateHandler.
func (h *ManifestIndexCreateHandler) Render() error {
	return h.printer.Println("Digest:", h.root.Digest)
}
