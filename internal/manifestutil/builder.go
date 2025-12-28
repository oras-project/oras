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

// Package manifestutil provides utilities for building OCI manifests and indexes.
package manifestutil

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"

	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras/internal/descriptor"
	"oras.land/oras/internal/dir"
)

// BuilderOptions configures the manifest builder behavior.
type BuilderOptions struct {
	// MaxBlobsPerManifest limits the number of layers per manifest.
	// If a directory has more files, they are split across multiple manifests
	// and combined in an index.
	MaxBlobsPerManifest int

	// ArtifactType is the artifact type for manifests.
	ArtifactType string

	// ManifestAnnotations are annotations to add to all manifests.
	ManifestAnnotations map[string]string

	// PreserveEmptyDirs includes empty directories as empty manifests.
	PreserveEmptyDirs bool
}

// BuildResult contains the result of building manifests for a directory tree.
type BuildResult struct {
	// Root is the root descriptor (index or manifest).
	Root ocispec.Descriptor
	// AllDescriptors contains all created descriptors (blobs, manifests, indexes).
	AllDescriptors []ocispec.Descriptor
	// ManifestCount is the number of manifests created.
	ManifestCount int
	// IndexCount is the number of indexes created.
	IndexCount int
}

// Builder builds hierarchical OCI manifests and indexes from a directory tree.
type Builder struct {
	opts   BuilderOptions
	pusher content.Pusher
}

// NewBuilder creates a new manifest builder.
func NewBuilder(pusher content.Pusher, opts BuilderOptions) *Builder {
	if opts.MaxBlobsPerManifest <= 0 {
		opts.MaxBlobsPerManifest = 1000
	}
	return &Builder{
		opts:   opts,
		pusher: pusher,
	}
}

// BuildFromNode builds manifests/indexes for a directory node and its children.
// File descriptors must already be pushed and provided in fileDescs map (path -> descriptor).
// Returns the root descriptor for this node.
func (b *Builder) BuildFromNode(ctx context.Context, node *dir.Node, fileDescs map[string]ocispec.Descriptor) (*BuildResult, error) {
	result := &BuildResult{}

	rootDesc, err := b.buildNode(ctx, node, fileDescs, result, true)
	if err != nil {
		return nil, err
	}

	result.Root = rootDesc
	return result, nil
}

// buildNode recursively builds manifests/indexes for a node.
func (b *Builder) buildNode(ctx context.Context, node *dir.Node, fileDescs map[string]ocispec.Descriptor, result *BuildResult, isRoot bool) (ocispec.Descriptor, error) {
	if !node.IsDir {
		// For files, just return the existing descriptor
		if desc, ok := fileDescs[node.Path]; ok {
			return desc, nil
		}
		return ocispec.Descriptor{}, nil
	}

	// Collect file descriptors for this directory
	var fileLayerDescs []ocispec.Descriptor
	for _, child := range node.Files() {
		if desc, ok := fileDescs[child.Path]; ok {
			fileLayerDescs = append(fileLayerDescs, desc)
		}
	}

	// Recursively build child directories
	var childDirDescs []ocispec.Descriptor
	for _, childDir := range node.Dirs() {
		childDesc, err := b.buildNode(ctx, childDir, fileDescs, result, false)
		if err != nil {
			return ocispec.Descriptor{}, err
		}
		if childDesc.Digest != "" {
			// Add directory annotations
			if childDesc.Annotations == nil {
				childDesc.Annotations = make(map[string]string)
			}
			maps.Copy(childDesc.Annotations, descriptor.MakeDirectoryAnnotations(childDir.Path, childDir.Name))
			childDirDescs = append(childDirDescs, childDesc)
		}
	}

	// Determine what to create based on contents
	hasFiles := len(fileLayerDescs) > 0
	hasDirs := len(childDirDescs) > 0
	isEmpty := !hasFiles && !hasDirs

	if isEmpty {
		if b.opts.PreserveEmptyDirs {
			// Create empty manifest for empty directory
			return b.createManifest(ctx, nil, node, result, isRoot)
		}
		return ocispec.Descriptor{}, nil
	}

	// Check if we need chunking for files
	needsChunking := len(fileLayerDescs) > b.opts.MaxBlobsPerManifest

	if !hasDirs && !needsChunking {
		// Simple case: only files, fits in one manifest
		return b.createManifest(ctx, fileLayerDescs, node, result, isRoot)
	}

	// Complex case: need an index
	var indexManifests []ocispec.Descriptor

	// Handle file chunks
	if hasFiles {
		chunks := chunkDescriptors(fileLayerDescs, b.opts.MaxBlobsPerManifest)
		for i, chunkDescs := range chunks {
			annotations := map[string]string{
				ocispec.AnnotationTitle: node.Name,
			}
			if len(chunks) > 1 {
				annotations["org.oras.content.chunk.index"] = string(rune('0' + i))
			}
			manifestDesc, err := b.createManifestWithAnnotations(ctx, chunkDescs, annotations, result)
			if err != nil {
				return ocispec.Descriptor{}, err
			}
			indexManifests = append(indexManifests, manifestDesc)
		}
	}

	// Add child directory indexes/manifests
	indexManifests = append(indexManifests, childDirDescs...)

	// Create the index
	return b.createIndex(ctx, indexManifests, node, result, isRoot)
}

// createManifest creates and pushes an OCI manifest.
func (b *Builder) createManifest(ctx context.Context, layers []ocispec.Descriptor, node *dir.Node, result *BuildResult, isRoot bool) (ocispec.Descriptor, error) {
	annotations := make(map[string]string)
	if node != nil {
		annotations[ocispec.AnnotationTitle] = node.Name
	}
	if isRoot {
		maps.Copy(annotations, descriptor.MakeRootAnnotations())
	}
	maps.Copy(annotations, b.opts.ManifestAnnotations)

	return b.createManifestWithAnnotations(ctx, layers, annotations, result)
}

// createManifestWithAnnotations creates and pushes an OCI manifest with specific annotations.
func (b *Builder) createManifestWithAnnotations(ctx context.Context, layers []ocispec.Descriptor, annotations map[string]string, result *BuildResult) (ocispec.Descriptor, error) {
	if layers == nil {
		layers = []ocispec.Descriptor{}
	}

	// Create empty config
	configBytes := []byte("{}")
	configDesc := content.NewDescriptorFromBytes(ocispec.MediaTypeImageConfig, configBytes)

	// Push config
	if err := b.pusher.Push(ctx, configDesc, bytes.NewReader(configBytes)); err != nil {
		// Ignore if already exists
		if !errors.Is(err, errdef.ErrAlreadyExists) {
			return ocispec.Descriptor{}, err
		}
	}

	manifest := ocispec.Manifest{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		MediaType: ocispec.MediaTypeImageManifest,
		Config:    configDesc,
		Layers:    layers,
	}
	if b.opts.ArtifactType != "" {
		manifest.ArtifactType = b.opts.ArtifactType
	}
	if len(annotations) > 0 {
		manifest.Annotations = annotations
	}

	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	manifestDesc := content.NewDescriptorFromBytes(ocispec.MediaTypeImageManifest, manifestBytes)
	manifestDesc.ArtifactType = b.opts.ArtifactType
	if len(annotations) > 0 {
		manifestDesc.Annotations = annotations
	}

	if err := b.pusher.Push(ctx, manifestDesc, bytes.NewReader(manifestBytes)); err != nil {
		if !errors.Is(err, errdef.ErrAlreadyExists) {
			return ocispec.Descriptor{}, err
		}
	}

	result.ManifestCount++
	result.AllDescriptors = append(result.AllDescriptors, manifestDesc)

	return manifestDesc, nil
}

// createIndex creates and pushes an OCI index.
func (b *Builder) createIndex(ctx context.Context, manifests []ocispec.Descriptor, node *dir.Node, result *BuildResult, isRoot bool) (ocispec.Descriptor, error) {
	annotations := make(map[string]string)
	if node != nil {
		annotations[ocispec.AnnotationTitle] = node.Name
		maps.Copy(annotations, descriptor.MakeDirectoryAnnotations(node.Path, node.Name))
	}
	if isRoot {
		maps.Copy(annotations, descriptor.MakeRootAnnotations())
	}
	maps.Copy(annotations, b.opts.ManifestAnnotations)

	index := ocispec.Index{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		MediaType: ocispec.MediaTypeImageIndex,
		Manifests: manifests,
	}
	if b.opts.ArtifactType != "" {
		index.ArtifactType = b.opts.ArtifactType
	}
	if len(annotations) > 0 {
		index.Annotations = annotations
	}

	indexBytes, err := json.Marshal(index)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	indexDesc := content.NewDescriptorFromBytes(ocispec.MediaTypeImageIndex, indexBytes)
	indexDesc.ArtifactType = b.opts.ArtifactType
	if len(annotations) > 0 {
		indexDesc.Annotations = annotations
	}

	if err := b.pusher.Push(ctx, indexDesc, bytes.NewReader(indexBytes)); err != nil {
		if !errors.Is(err, errdef.ErrAlreadyExists) {
			return ocispec.Descriptor{}, err
		}
	}

	result.IndexCount++
	result.AllDescriptors = append(result.AllDescriptors, indexDesc)

	return indexDesc, nil
}

// chunkDescriptors splits a list of descriptors into chunks of at most maxSize.
func chunkDescriptors(descs []ocispec.Descriptor, maxSize int) [][]ocispec.Descriptor {
	if maxSize <= 0 || len(descs) <= maxSize {
		return [][]ocispec.Descriptor{descs}
	}

	var chunks [][]ocispec.Descriptor
	for i := 0; i < len(descs); i += maxSize {
		end := i + maxSize
		if end > len(descs) {
			end = len(descs)
		}
		chunks = append(chunks, descs[i:end])
	}
	return chunks
}
