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

package content

import (
	"fmt"
	"io"
	"os"

	"oras.land/oras/cmd/oras/internal/output"
)

// manifestIndexCreate handles raw content output.
type manifestIndexCreate struct {
	pretty     bool
	stdout     io.Writer
	outputPath string
}

// NewManifestIndexCreateHandler creates a new handler.
func NewManifestIndexCreateHandler(out io.Writer, pretty bool, outputPath string) ManifestIndexCreateHandler {
	// ignore --pretty when output to a file
	if outputPath != "" && outputPath != "-" {
		pretty = false
	}
	return &manifestIndexCreate{
		pretty:     pretty,
		stdout:     out,
		outputPath: outputPath,
	}
}

// OnContentCreated is called after index content is created.
func (h *manifestIndexCreate) OnContentCreated(manifest []byte) (eventErr error) {
	out := h.stdout
	if h.outputPath != "" && h.outputPath != "-" {
		f, err := os.Create(h.outputPath)
		if err != nil {
			return fmt.Errorf("failed to open %q: %w", h.outputPath, err)
		}
		defer func() {
			if err := f.Close(); eventErr == nil {
				eventErr = err
			}
		}()
		out = f
	}
	return output.PrintJSON(out, manifest, h.pretty)
}
