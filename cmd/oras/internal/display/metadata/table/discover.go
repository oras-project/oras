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
	"strings"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/json"
)

// discoverHandler handles json metadata output for discover events.
type discoverHandler struct {
	verbose          bool
	subjectReference string
	subjectDigest    digest.Digest
}

// OnDiscovered implements metadata.DiscoverHandler.
func (d discoverHandler) OnDiscovered(refs []v1.Descriptor) error {
	if n := len(refs); n > 1 {
		fmt.Println("Discovered", n, "artifacts referencing", d.subjectReference)
	} else {
		fmt.Println("Discovered", n, "artifact referencing", d.subjectReference)
	}
	fmt.Println("Digest:", d.subjectDigest)
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
func NewDiscoverHandler(reference string, digest digest.Digest, verbose bool) metadata.DiscoverHandler {
	return discoverHandler{
		verbose:          verbose,
		subjectReference: reference,
		subjectDigest:    digest,
	}
}
