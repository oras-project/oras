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

package graph

import (
	"context"
	"encoding/json"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"oras.land/oras/internal/docker"
)

// Successors returns the nodes directly pointed by the current node, picking
// out subject and config descriptor if applicable.
// Returning nil when no subject and config found.
func Successors(ctx context.Context, fetcher content.Fetcher, node ocispec.Descriptor) ([]ocispec.Descriptor, *ocispec.Descriptor, *ocispec.Descriptor, error) {
	var fetched []byte
	var nodes []ocispec.Descriptor
	var subject *ocispec.Descriptor
	var config *ocispec.Descriptor
	var err error
	switch node.MediaType {
	case docker.MediaTypeManifest, ocispec.MediaTypeImageManifest:
		fetched, err = content.FetchAll(ctx, fetcher, node)
		if err != nil {
			return nil, nil, nil, err
		}
		var manifest ocispec.Manifest
		if err = json.Unmarshal(fetched, &manifest); err != nil {
			return nil, nil, nil, err
		}
		subject = manifest.Subject
		config = &manifest.Config
		nodes = manifest.Layers
	case ocispec.MediaTypeArtifactManifest:
		fetched, err = content.FetchAll(ctx, fetcher, node)
		if err != nil {
			return nil, nil, nil, err
		}
		var manifest ocispec.Artifact
		if err = json.Unmarshal(fetched, &manifest); err != nil {
			return nil, nil, nil, err
		}
		subject = manifest.Subject
		nodes = manifest.Blobs
	default:
		nodes, err = content.Successors(ctx, fetcher, node)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	return nodes, subject, config, nil
}
