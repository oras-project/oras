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

// BackupHandler handles text metadata output for cp events.
type BackupHandler struct {
	printer *output.Printer
}

// NewBackupHandler returns a new handler for cp events.
func NewBackupHandler(printer *output.Printer) metadata.BackupHandler {
	return &BackupHandler{
		printer: printer,
	}
}

// OnTagged implements metadata.TaggedHandler.
func (h *BackupHandler) OnTagged(_ ocispec.Descriptor, tag string) error {
	return h.printer.Println("Tagged", tag)
}

// OnCopied implements metadata.BackupHandler.
func (h *BackupHandler) OnCopied(source, destination string) error {
	return h.printer.Println("Copied", source, "=>", destination)
}
