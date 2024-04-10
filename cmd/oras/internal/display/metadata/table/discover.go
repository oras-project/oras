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

package table

import (
	"fmt"
	"io"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/json"
	"oras.land/oras/internal/registryutil"
)

// discoverHandler handles json metadata output for discover events.
type discoverHandler struct {
	referrers    registryutil.ReferrersFunc
	template     string
	path         string
	desc         ocispec.Descriptor
	rawReference string
	verbose      bool
	out          io.Writer
}

// OnDiscovered implements metadata.DiscoverHandler.
func (h *discoverHandler) OnDiscovered() error {
	refs, err := h.referrers(h.desc)
	if err != nil {
		return err
	}
	if n := len(refs); n > 1 {
		fmt.Println("Discovered", n, "artifacts referencing", h.rawReference)
	} else {
		fmt.Println("Discovered", n, "artifact referencing", h.rawReference)
	}
	fmt.Println("Digest:", h.desc.Digest)
	if len(refs) > 0 {
		fmt.Println()
		return h.printDiscoveredReferrersTable(refs, h.verbose)
	}
	return nil
}

func (h *discoverHandler) printDiscoveredReferrersTable(refs []ocispec.Descriptor, verbose bool) error {
	typeNameTitle := "Artifact Type"
	typeNameLength := len(typeNameTitle)
	for _, ref := range refs {
		if length := len(ref.ArtifactType); length > typeNameLength {
			typeNameLength = length
		}
	}

	print := func(key string, value interface{}) {
		fmt.Println(key, strings.Repeat(" ", typeNameLength-len(key)+1), value)
	}

	print(typeNameTitle, "Digest")
	for _, ref := range refs {
		print(ref.ArtifactType, ref.Digest)
		if verbose {
			if err := json.PrintJSON(h.out, ref); err != nil {
				return fmt.Errorf("error printing JSON: %w", err)
			}
		}
	}
	return nil
}

// NewDiscoverHandler creates a new handler for discover events.
func NewDiscoverHandler(out io.Writer, template string, path string, desc ocispec.Descriptor, referrers registryutil.ReferrersFunc, rawReference string, verbose bool) metadata.DiscoverHandler {
	return &discoverHandler{
		template:     template,
		path:         path,
		desc:         desc,
		rawReference: rawReference,
		verbose:      verbose,
		out:          out,
		referrers:    referrers,
	}
}
