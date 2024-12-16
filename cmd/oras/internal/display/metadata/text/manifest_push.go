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
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
)

// ManifestPushHandler handles text metadata output for manifest push events.
type ManifestPushHandler struct {
	printer *output.Printer
	target  *option.Target
	desc    ocispec.Descriptor
}

// NewManifestPushHandler returns a new handler for manifest push events.
func NewManifestPushHandler(printer *output.Printer, target *option.Target) metadata.ManifestPushHandler {
	return &ManifestPushHandler{
		printer: printer,
		target:  target,
	}
}

// OnTagged implements metadata.TaggedHandler.
func (h *ManifestPushHandler) OnTagged(_ ocispec.Descriptor, tag string) error {
	return h.printer.Println("Tagged", tag)
}

// OnManifestPushed implements metadata.ManifestPushHandler.
func (h *ManifestPushHandler) OnManifestPushed(desc ocispec.Descriptor) error {
	h.desc = desc
	return h.printer.Println("Pushed:", h.target.AnnotatedReference())
}

// Render implements metadata.ManifestPushHandler.
func (h *ManifestPushHandler) Render() error {
	return h.printer.Println("Digest:", h.desc.Digest)
}
