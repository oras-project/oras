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
	"context"
	"encoding/json"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"oras.land/oras/internal/docker"
)

// Successors returns the nodes directly pointed by the current node, as well as
// subject and config descriptor if applicable.
func Successors(ctx context.Context, fetcher content.Fetcher, node ocispec.Descriptor) (nodes []ocispec.Descriptor, subject ocispec.Descriptor, config ocispec.Descriptor, err error) {
	var fetched []byte
	switch node.MediaType {
	case ocispec.MediaTypeImageManifest, docker.MediaTypeManifest:
		fetched, err = content.FetchAll(ctx, fetcher, node)
		if err != nil {
			return
		}
		var manifest ocispec.Manifest
		if err = json.Unmarshal(fetched, &manifest); err != nil {
			return
		}
		if manifest.Subject != nil {
			subject = *manifest.Subject
			nodes = append(nodes, subject)
		}
		config = manifest.Config
		nodes = append(append(manifest.Layers, config), nodes...)
	case ocispec.MediaTypeArtifactManifest:
		fetched, err = content.FetchAll(ctx, fetcher, node)
		if err != nil {
			return
		}
		var manifest ocispec.Artifact
		if err = json.Unmarshal(fetched, &manifest); err != nil {
			return
		}
		if manifest.Subject != nil {
			subject = *manifest.Subject
			nodes = append(nodes, subject)
		}
		nodes = append(nodes, manifest.Blobs...)
	default:
		nodes, err = content.Successors(ctx, fetcher, node)
	}
	return
}
