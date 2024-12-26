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

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/output"
)

// manifestFetch handles raw content output.
type manifestFetch struct {
	pretty     bool
	stdout     io.Writer
	outputPath string
}

func (h *manifestFetch) OnContentFetched(desc ocispec.Descriptor, manifest []byte) (eventErr error) {
	out := h.stdout
	if h.outputPath != "-" && h.outputPath != "" {
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

// NewManifestFetchHandler creates a new handler.
func NewManifestFetchHandler(out io.Writer, pretty bool, outputPath string) ManifestFetchHandler {
	// ignore --pretty when output to a file
	if outputPath != "" && outputPath != "-" {
		pretty = false
	}
	return &manifestFetch{
		pretty:     pretty,
		stdout:     out,
		outputPath: outputPath,
	}
}
