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
	"oras.land/oras-go/v2/content"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/output"
)

// discoverHandler handles json metadata output for discover events.
type discoverHandler struct {
	out          io.Writer
	rawReference string
	root         ocispec.Descriptor
	verbose      bool
	referrers    []ocispec.Descriptor
}

// NewDiscoverHandler creates a new handler for discover events.
func NewDiscoverHandler(out io.Writer, rawReference string, root ocispec.Descriptor, verbose bool) metadata.DiscoverHandler {
	return &discoverHandler{
		out:          out,
		rawReference: rawReference,
		root:         root,
		verbose:      verbose,
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
	if n := len(h.referrers); n > 1 {
		fmt.Fprintln(h.out, "Discovered", n, "artifacts referencing", h.rawReference)
	} else {
		fmt.Fprintln(h.out, "Discovered", n, "artifact referencing", h.rawReference)
	}
	fmt.Fprintln(h.out, "Digest:", h.root.Digest)
	if len(h.referrers) == 0 {
		return nil
	}
	fmt.Fprintln(h.out)
	return h.printDiscoveredReferrersTable()
}

func (h *discoverHandler) printDiscoveredReferrersTable() error {
	typeNameTitle := "Artifact Type"
	typeNameLength := len(typeNameTitle)
	for _, ref := range h.referrers {
		if length := len(ref.ArtifactType); length > typeNameLength {
			typeNameLength = length
		}
	}

	print := func(key string, value interface{}) {
		fmt.Fprintln(h.out, key, strings.Repeat(" ", typeNameLength-len(key)+1), value)
	}

	print(typeNameTitle, "Digest")
	for _, ref := range h.referrers {
		print(ref.ArtifactType, ref.Digest)
		if h.verbose {
			if err := output.PrintPrettyJSON(h.out, ref); err != nil {
				return fmt.Errorf("error printing JSON: %w", err)
			}
		}
	}
	return nil
}
