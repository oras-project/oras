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
	"oras.land/oras-go/v2/registry"
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

// Referrers returns referrer nodes of desc in target.
func Referrers(ctx context.Context, target content.ReadOnlyGraphStorage, desc ocispec.Descriptor, artifactType string) ([]ocispec.Descriptor, error) {
	var results []ocispec.Descriptor
	if repo, ok := target.(registry.ReferrerLister); ok {
		// get referrers directly
		err := repo.Referrers(ctx, desc, artifactType, func(referrers []ocispec.Descriptor) error {
			results = append(results, referrers...)
			return nil
		})
		if err != nil {
			return nil, err
		}
		return results, nil
	}

	// find matched referrers in all predecessors
	predecessors, err := target.Predecessors(ctx, desc)
	if err != nil {
		return nil, err
	}
	for _, node := range predecessors {
		switch node.MediaType {
		case MediaTypeArtifactManifest:
			fetched, err := fetchBytes(ctx, target, node)
			if err != nil {
				return nil, err
			}
			var artifact Artifact
			if err := json.Unmarshal(fetched, &artifact); err != nil {
				return nil, err
			}
			if artifact.Subject == nil || !content.Equal(*artifact.Subject, desc) {
				continue
			}
			node.ArtifactType = artifact.ArtifactType
			node.Annotations = artifact.Annotations
		case ocispec.MediaTypeImageManifest:
			fetched, err := fetchBytes(ctx, target, node)
			if err != nil {
				return nil, err
			}
			var image ocispec.Manifest
			if err := json.Unmarshal(fetched, &image); err != nil {
				return nil, err
			}
			if image.Subject == nil || !content.Equal(*image.Subject, desc) {
				continue
			}
			node.ArtifactType = image.ArtifactType
			if node.ArtifactType == "" {
				node.ArtifactType = image.Config.MediaType
			}
			node.Annotations = image.Annotations
		case ocispec.MediaTypeImageIndex:
			fetched, err := fetchBytes(ctx, target, node)
			if err != nil {
				return nil, err
			}
			var index ocispec.Index
			if err := json.Unmarshal(fetched, &index); err != nil {
				return nil, err
			}
			if index.Subject == nil || !content.Equal(*index.Subject, desc) {
				continue
			}
			node.ArtifactType = index.ArtifactType
			node.Annotations = index.Annotations
		default:
			continue
		}
		if artifactType == "" || artifactType == node.ArtifactType {
			// the field artifactType in referrers descriptor is allowed to be empty
			// https://github.com/opencontainers/distribution-spec/issues/458
			results = append(results, node)
		}
	}
	return results, nil
}

func fetchBytes(ctx context.Context, fetcher content.Fetcher, desc ocispec.Descriptor) ([]byte, error) {
	rc, err := fetcher.Fetch(ctx, desc)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return content.ReadAll(rc, desc)
}

// FindPredecessorsCurrently returns all predecessors of descs in src concurrently.
func FindPredecessorsCurrently(ctx context.Context, src oras.ReadOnlyGraphTarget, descs []ocispec.Descriptor, opts oras.ExtendedCopyOptions) ([]ocispec.Descriptor, error) {
	// point referrers of child manifests to root
	var referrers []ocispec.Descriptor
	g, ctx := errgroup.WithContext(ctx)
	var m sync.Mutex
	g.SetLimit(opts.Concurrency)
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
