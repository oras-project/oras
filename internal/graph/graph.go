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
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"golang.org/x/sync/errgroup"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras/internal/docker"
)

// MediaTypeArtifactManifest specifies the media type for a content descriptor.
const MediaTypeArtifactManifest = "application/vnd.oci.artifact.manifest.v1+json"

// Artifact describes an artifact manifest.
// This structure provides `application/vnd.oci.artifact.manifest.v1+json` mediatype when marshalled to JSON.
//
// This manifest type was introduced in image-spec v1.1.0-rc1 and was removed in
// image-spec v1.1.0-rc3. It is not part of the current image-spec and is kept
// here for Go compatibility.
//
// Reference: https://github.com/opencontainers/image-spec/pull/999
type Artifact struct {
	// MediaType is the media type of the object this schema refers to.
	MediaType string `json:"mediaType"`

	// ArtifactType is the IANA media type of the artifact this schema refers to.
	ArtifactType string `json:"artifactType"`

	// Blobs is a collection of blobs referenced by this manifest.
	Blobs []ocispec.Descriptor `json:"blobs,omitempty"`

	// Subject (reference) is an optional link from the artifact to another manifest forming an association between the artifact and the other manifest.
	Subject *ocispec.Descriptor `json:"subject,omitempty"`

	// Annotations contains arbitrary metadata for the artifact manifest.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Successors returns the nodes directly pointed by the current node, picking
// out subject and config descriptor if applicable.
// Returning nil when no subject and config found.
func Successors(ctx context.Context, fetcher content.Fetcher, node ocispec.Descriptor) (nodes []ocispec.Descriptor, subject, config *ocispec.Descriptor, err error) {
	switch node.MediaType {
	case docker.MediaTypeManifest, ocispec.MediaTypeImageManifest:
		var fetched []byte
		fetched, err = content.FetchAll(ctx, fetcher, node)
		if err != nil {
			return
		}
		var manifest ocispec.Manifest
		if err = json.Unmarshal(fetched, &manifest); err != nil {
			return
		}
		nodes = manifest.Layers
		subject = manifest.Subject
		config = &manifest.Config
	case MediaTypeArtifactManifest:
		var fetched []byte
		fetched, err = content.FetchAll(ctx, fetcher, node)
		if err != nil {
			return
		}
		var manifest Artifact
		if err = json.Unmarshal(fetched, &manifest); err != nil {
			return
		}
		nodes = manifest.Blobs
		subject = manifest.Subject
	case ocispec.MediaTypeImageIndex:
		var fetched []byte
		fetched, err = content.FetchAll(ctx, fetcher, node)
		if err != nil {
			return
		}
		var index ocispec.Index
		if err = json.Unmarshal(fetched, &index); err != nil {
			return
		}
		nodes = index.Manifests
		subject = index.Subject
	default:
		nodes, err = content.Successors(ctx, fetcher, node)
	}
	return
}

// FindPredecessors returns all predecessors of descs in src concurrently.
func FindPredecessors(ctx context.Context, src oras.ReadOnlyGraphTarget, descs []ocispec.Descriptor, opts oras.ExtendedCopyOptions) ([]ocispec.Descriptor, error) {
	var referrers []ocispec.Descriptor
	g, ctx := errgroup.WithContext(ctx)
	var m sync.Mutex
	if opts.Concurrency != 0 {
		g.SetLimit(opts.Concurrency)
	}
	for _, desc := range descs {
		g.Go(func(node ocispec.Descriptor) func() error {
			return func() error {
				descs, err := opts.FindPredecessors(ctx, src, node)
				if err != nil {
					return err
				}
				m.Lock()
				defer m.Unlock()
				referrers = append(referrers, descs...)
				return nil
			}
		}(desc))
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return referrers, nil
}
