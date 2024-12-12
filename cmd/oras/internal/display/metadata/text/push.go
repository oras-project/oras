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
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
)

// PushHandler handles text metadata output for push events.
type PushHandler struct {
	printer *output.Printer
	tagLock sync.Mutex
	root    ocispec.Descriptor
}

// NewPushHandler returns a new handler for push events.
func NewPushHandler(printer *output.Printer) metadata.PushHandler {
	return &PushHandler{
		printer: printer,
	}
}

// OnTagged implements metadata.TaggedHandler.
func (h *PushHandler) OnTagged(_ ocispec.Descriptor, tag string) error {
	h.tagLock.Lock()
	defer h.tagLock.Unlock()
	return h.printer.Println("Tagged", tag)
}

// OnCopied is called after files are copied.
func (h *PushHandler) OnCopied(opts *option.Target, root ocispec.Descriptor) error {
	h.root = root
	return h.printer.Println("Pushed", opts.AnnotatedReference())
}

// Render implements PushHandler.
func (h *PushHandler) Render() error {
	err := h.printer.Println("ArtifactType:", h.root.ArtifactType)
	if err != nil {
		return err
	}
	return h.printer.Println("Digest:", h.root.Digest)
}
