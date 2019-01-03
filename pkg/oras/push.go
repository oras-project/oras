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
func Push(ctx context.Context, resolver remotes.Resolver, ref string, provider content.Provider, descriptors []ocispec.Descriptor) error {
	if resolver == nil {
		return ErrResolverUndefined
	}

	if descriptors == nil {
		return ErrEmptyDescriptors
	}

	pusher, err := resolver.Pusher(ctx, ref)
	if err != nil {
		return err
	}

	desc, provider, err := pack(provider, descriptors)
	if err != nil {
		return err
	}

	return remotes.PushContent(ctx, pusher, desc, provider, nil)
}

func pack(provider content.Provider, descriptors []ocispec.Descriptor) (ocispec.Descriptor, content.Provider, error) {
	store := newHybridStoreFromProvider(provider)

	// Config
	configBytes := []byte("{}")
	config := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageConfig,
		Digest:    digest.FromBytes(configBytes),
		Size:      int64(len(configBytes)),
	}
	store.Set(config, configBytes)

	// Manifest
	manifest := ocispec.Manifest{
		Versioned: specs.Versioned{
			SchemaVersion: 2, // historical value. does not pertain to OCI or docker version
		},
		Config: config,
		Layers: descriptors,
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
