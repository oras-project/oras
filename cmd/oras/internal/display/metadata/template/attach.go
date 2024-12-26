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

package template

import (
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/model"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
)

// AttachHandler handles go-template metadata output for attach events.
type AttachHandler struct {
	template string
	out      io.Writer
	path     string
	root     ocispec.Descriptor
}

// NewAttachHandler returns a new handler for attach metadata events.
func NewAttachHandler(out io.Writer, template string) metadata.AttachHandler {
	return &AttachHandler{
		out:      out,
		template: template,
	}
}

// OnAttached implements AttachHandler.
func (ah *AttachHandler) OnAttached(target *option.Target, root ocispec.Descriptor, _ ocispec.Descriptor) {
	ah.path = target.Path
	ah.root = root
}

// Render formats the metadata of attach command.
func (ah *AttachHandler) Render() error {
	return output.ParseAndWrite(ah.out, model.NewAttach(ah.root, ah.path), ah.template)
}
