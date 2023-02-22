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
	"oras.land/oras-go/v2/registry"
	"oras.land/oras/internal/docker"
)

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
	case ocispec.MediaTypeArtifactManifest:
		var fetched []byte
		fetched, err = content.FetchAll(ctx, fetcher, node)
		if err != nil {
			return
		}
		var manifest ocispec.Artifact
		if err = json.Unmarshal(fetched, &manifest); err != nil {
			return
		}
		nodes = manifest.Blobs
		subject = manifest.Subject
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
		case ocispec.MediaTypeArtifactManifest:
			fetched, err := fetchBytes(ctx, target, node)
			if err != nil {
				return nil, err
			}
			var artifact ocispec.Artifact
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
			node.ArtifactType = image.Config.MediaType
			node.Annotations = image.Annotations
		default:
			continue
		}
		if node.ArtifactType != "" && (artifactType == "" || artifactType == node.ArtifactType) {
			results = append(results, node)
		}
	}
	return results, nil
}

// FindReferrerPredecessors returns referrer nodes of desc in target.
func FindReferrerPredecessors(ctx context.Context, src content.ReadOnlyGraphStorage, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
	var results []ocispec.Descriptor
	if repo, ok := src.(registry.ReferrerLister); ok {
		// get referrers directly
		err := repo.Referrers(ctx, desc, "", func(referrers []ocispec.Descriptor) error {
			results = append(results, referrers...)
			return nil
		})
		if err != nil {
			return nil, err
		}
		return results, nil
	}
	predecessors, err := src.Predecessors(ctx, desc)
	if err != nil {
		return nil, err
	}
	for _, node := range predecessors {
		switch node.MediaType {
		case ocispec.MediaTypeArtifactManifest, ocispec.MediaTypeImageManifest:
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
