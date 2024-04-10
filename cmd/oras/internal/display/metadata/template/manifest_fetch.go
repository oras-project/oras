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
	"encoding/json"
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/model"
)

// manifestFetchHandler handles JSON metadata output for manifest fetch events.
type manifestFetchHandler struct {
	template string
	out      io.Writer
}

// NewManifestFetchHandler creates a new handler for manifest fetch events.
func NewManifestFetchHandler(out io.Writer, template string) metadata.ManifestFetchHandler {
	return &manifestFetchHandler{
		template: template,
		out:      out,
	}
}

// OnFetched is called after the manifest fetch is completed.
func (h *manifestFetchHandler) OnFetched(path string, desc ocispec.Descriptor, content []byte) error {
	var manifest map[string]any
	_ = json.Unmarshal(content, &manifest)
	return parseAndWrite(h.out, model.NewFetched(path, desc, manifest), h.template)
}
