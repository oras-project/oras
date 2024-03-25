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
	"context"
	"fmt"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/json"
)

// discoverHandler handles json metadata output for discover events.
type discoverHandler struct {
	ctx          context.Context
	repo         oras.ReadOnlyGraphTarget
	template     string
	path         string
	desc         ocispec.Descriptor
	artifactType string
	rawReference string
	verbose      bool
}

// OnDiscovered implements metadata.DiscoverHandler.
func (d discoverHandler) OnDiscovered() error {
	refs, err := registry.Referrers(d.ctx, d.repo, d.desc, d.artifactType)
	if err != nil {
		return err
	}
	if n := len(refs); n > 1 {
		fmt.Println("Discovered", n, "artifacts referencing", d.rawReference)
	} else {
		fmt.Println("Discovered", n, "artifact referencing", d.rawReference)
	}
	fmt.Println("Digest:", d.desc.Digest)
	if len(refs) > 0 {
		fmt.Println()
		return printDiscoveredReferrersTable(refs, d.verbose)
	}
	return nil
}

func printDiscoveredReferrersTable(refs []ocispec.Descriptor, verbose bool) error {
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
			if err := json.PrintJSON(ref); err != nil {
				return fmt.Errorf("error printing JSON: %w", err)
			}
		}
	}
	return nil
}

// NewDiscoverHandler creates a new handler for discover events.
func NewDiscoverHandler(ctx context.Context, template string, path string, artifactType string, desc ocispec.Descriptor, repo oras.ReadOnlyGraphTarget, rawReference string, verbose bool) metadata.DiscoverHandler {
	return discoverHandler{
		template:     template,
		path:         path,
		ctx:          ctx,
		repo:         repo,
		desc:         desc,
		artifactType: artifactType,
		rawReference: rawReference,
		verbose:      verbose,
	}
}
