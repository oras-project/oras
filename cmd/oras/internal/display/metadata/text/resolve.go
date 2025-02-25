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

// ResolveHandler handles text metadata output for resolve events.
type ResolveHandler struct {
	printer *output.Printer
	fullRef bool
	path    string
}

// NewResolveHandler returns a new handler for resolve events.
func NewResolveHandler(printer *output.Printer, fullRef bool, path string) metadata.ResolveHandler {
	return &ResolveHandler{
		printer: printer,
		fullRef: fullRef,
		path:    path,
	}
}

// OnResolved implements metadata.ResolveHandler.
func (h *ResolveHandler) OnResolved(desc ocispec.Descriptor) error {
	if h.fullRef {
		return h.printer.Printf("%s@%s\n", h.path, desc.Digest)
	}
	return h.printer.Println(desc.Digest.String())
}
