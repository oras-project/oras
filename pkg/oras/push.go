package oras

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/containerd/containerd/remotes"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Push pushes files to the remote
func Push(ctx context.Context, resolver remotes.Resolver, ref string, contents map[string][]byte) error {
	if resolver == nil {
		return errors.New("resolver is undefined")
	}

	pusher, err := resolver.Pusher(ctx, ref)
	if err != nil {
		return err
	}
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
			SchemaVersion: 1,
		},
		Config: config,
		Layers: layers,
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	manifestDescriptor := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Digest:    digest.FromBytes(manifestBytes),
		Size:      int64(len(manifestBytes)),
	}
	store.Set(manifestDescriptor, manifestBytes)

	return remotes.PushContent(ctx, pusher, manifestDescriptor, store, nil)
}
