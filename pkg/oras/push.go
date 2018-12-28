package oras

import (
	"context"
	"encoding/json"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/remotes"
	digest "github.com/opencontainers/go-digest"
	specs "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Push pushes files to the remote
func Push(ctx context.Context, resolver remotes.Resolver, ref string, contents map[string][]byte) error {
	if resolver == nil {
		return ErrResolverUndefined
	}

	if contents == nil {
		return ErrEmptyContents
	}

	pusher, err := resolver.Pusher(ctx, ref)
	if err != nil {
		return err
	}

	desc, provider, err := pack(contents)
	if err != nil {
		return err
	}

	return remotes.PushContent(ctx, pusher, desc, provider, nil)
}

func pack(contents map[string][]byte) (ocispec.Descriptor, content.Provider, error) {
	store := NewMemoryStore()

	// Config
	configBytes := []byte("{}")
	config := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageConfig,
		Digest:    digest.FromBytes(configBytes),
		Size:      int64(len(configBytes)),
	}
	store.Set(config, configBytes)

	// Layer
	var layers []ocispec.Descriptor
	for name, content := range contents {
		layer := ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageLayer,
			Digest:    digest.FromBytes(content),
			Size:      int64(len(content)),
			Annotations: map[string]string{
				ocispec.AnnotationTitle: name,
			},
		}
		store.Set(layer, content)
		layers = append(layers, layer)
	}

	// Manifest
	manifest := ocispec.Manifest{
		Versioned: specs.Versioned{
			SchemaVersion: 2, // historical value. does not pertain to OCI or docker version
		},
		Config: config,
		Layers: layers,
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return ocispec.Descriptor{}, nil, err
	}
	manifestDescriptor := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Digest:    digest.FromBytes(manifestBytes),
		Size:      int64(len(manifestBytes)),
	}
	store.Set(manifestDescriptor, manifestBytes)

	return manifestDescriptor, store, nil
}
