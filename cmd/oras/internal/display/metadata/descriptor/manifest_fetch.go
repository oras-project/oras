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

package descriptor

import (
	"encoding/json"
	"fmt"
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/utils"
)

// manifestFetchHandler handles metadata descriptor output.
type manifestFetchHandler struct {
	pretty bool
	out    io.Writer
}

// OnFetched implements ManifestFetchHandler.
func (h *manifestFetchHandler) OnFetched(_ string, desc ocispec.Descriptor, _ []byte) error {
	descBytes, err := json.Marshal(desc)
	if err != nil {
		return fmt.Errorf("invalid descriptor: %w", err)
	}
	return utils.PrintJSON(h.out, descBytes, h.pretty)
}

// NewManifestFetchHandler creates a new handler.
func NewManifestFetchHandler(out io.Writer, pretty bool) metadata.ManifestFetchHandler {
	return &manifestFetchHandler{
		pretty: pretty,
		out:    out,
	}
}
