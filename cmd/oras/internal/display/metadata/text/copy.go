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

// CopyHandler handles text metadata output for cp events.
type CopyHandler struct {
	printer *output.Printer
	desc    ocispec.Descriptor
}

// NewCopyHandler returns a new handler for cp events.
func NewCopyHandler(printer *output.Printer) metadata.CopyHandler {
	return &CopyHandler{
		printer: printer,
	}
}

// OnTagged implements metadata.TaggedHandler.
func (h *CopyHandler) OnTagged(_ ocispec.Descriptor, tag string) error {
	return h.printer.Println("Tagged", tag)
}

// Render implements metadata.Renderer.
func (h *CopyHandler) Render() error {
	return h.printer.Println("Digest:", h.desc.Digest)
}

// OnCopied implements metadata.CopyHandler.
func (h *CopyHandler) OnCopied(target *option.BinaryTarget, desc ocispec.Descriptor) error {
	h.desc = desc
	return h.printer.Println("Copied", target.From.GetDisplayReference(), "=>", target.To.GetDisplayReference())
}
