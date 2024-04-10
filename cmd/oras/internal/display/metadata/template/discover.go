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
	"oras.land/oras/internal/registryutil"
)

// DiscoverHandler handles json metadata output for discover events.
type DiscoverHandler struct {
	referrers registryutil.ReferrersFunc
	template  string
	path      string
	desc      ocispec.Descriptor
	out       io.Writer
}

// OnDiscovered implements metadata.DiscoverHandler.
func (h *DiscoverHandler) OnDiscovered() error {
	refs, err := h.referrers(h.desc)
	if err != nil {
		return err
	}
	return parseAndWrite(h.out, model.NewDiscover(h.path, refs), h.template)
}

// NewDiscoverHandler creates a new handler for discover events.
func NewDiscoverHandler(out io.Writer, template string, path string, desc ocispec.Descriptor, referrers registryutil.ReferrersFunc) metadata.DiscoverHandler {
	return &DiscoverHandler{
		template:  template,
		path:      path,
		referrers: referrers,
		desc:      desc,
		out:       out,
	}
}
