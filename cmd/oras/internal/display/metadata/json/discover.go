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

package json

import (
	"fmt"
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/model"
	"oras.land/oras/cmd/oras/internal/output"
)

// discoverHandler handles json metadata output for discover events.
type discoverHandler struct {
	out       io.Writer
	root      ocispec.Descriptor
	path      string
	referrers []ocispec.Descriptor
}

// NewDiscoverHandler creates a new handler for discover events.
func NewDiscoverHandler(out io.Writer, root ocispec.Descriptor, path string) metadata.DiscoverHandler {
	return &discoverHandler{
		out:  out,
		root: root,
		path: path,
	}
}

// MultiLevelSupported implements metadata.DiscoverHandler.
func (h *discoverHandler) MultiLevelSupported() bool {
	return false
}

// OnDiscovered implements metadata.DiscoverHandler.
func (h *discoverHandler) OnDiscovered(referrer, subject ocispec.Descriptor) error {
	if !content.Equal(subject, h.root) {
		return fmt.Errorf("unexpected subject descriptor: %v", subject)
	}
	h.referrers = append(h.referrers, referrer)
	return nil
}

// Render implements metadata.DiscoverHandler.
func (h *discoverHandler) Render() error {
	return output.PrintPrettyJSON(h.out, model.NewDiscover(h.path, h.referrers))
}
