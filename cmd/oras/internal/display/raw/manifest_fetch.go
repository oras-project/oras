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

package raw

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// RawManifestFetch handles raw content output.
type RawManifestFetch struct {
	pretty bool
	stdout io.Writer
}

// OnContentFetched implements ManifestFetchHandler.
func (h *RawManifestFetch) OnContentFetched(outputPath string, manifest []byte) error {
	out := h.stdout
	if outputPath != "-" && outputPath != "" {
		f, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			return fmt.Errorf("failed to open %q: %w", outputPath, err)
		}
		defer f.Close()
	}
	return h.output(out, manifest)
}

// OnDescriptorFetched implements ManifestFetchHandler.
func (h *RawManifestFetch) OnDescriptorFetched(desc ocispec.Descriptor) error {
	descBytes, err := json.Marshal(desc)
	if err != nil {
		return fmt.Errorf("invalid descriptor: %w", err)
	}
	return h.output(h.stdout, descBytes)
}

// NewManifestFetchHandler creates a new handler.
func NewManifestFetchHandler(out io.Writer, pretty bool) ManifestFetchHandler {
	return &RawManifestFetch{
		pretty: pretty,
		stdout: out,
	}
}

// OnFetched is called after the content is fetched.
func (h *RawManifestFetch) output(out io.Writer, data []byte) error {
	if h.pretty {
		buf := bytes.NewBuffer(nil)
		if err := json.Indent(buf, data, "", "  "); err != nil {
			return fmt.Errorf("failed to prettify: %w", err)
		}
		buf.WriteByte('\n')
		data = buf.Bytes()
	}
	_, err := out.Write(data)
	return err
}
