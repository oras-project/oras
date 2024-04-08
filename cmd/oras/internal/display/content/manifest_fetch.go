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
	"oras.land/oras/cmd/oras/internal/display/utils"
)

// RawManifestFetch handles raw content output.
type RawManifestFetch struct {
	pretty     bool
	stdout     io.Writer
	outputPath string
}

func (h *RawManifestFetch) OnContentFetched(desc ocispec.Descriptor, manifest []byte) error {
	out := h.stdout
	if h.outputPath != "-" && h.outputPath != "" {
		f, err := os.OpenFile(h.outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			return fmt.Errorf("failed to open %q: %w", h.outputPath, err)
		}
		defer f.Close()
		out = f
	}
	return utils.Output(out, manifest, h.pretty)
}

// NewManifestFetchHandler creates a new handler.
func NewManifestFetchHandler(out io.Writer, pretty bool, outputPath string) ManifestFetchHandler {
	return &RawManifestFetch{
		pretty:     pretty,
		stdout:     out,
		outputPath: outputPath,
	}
}
