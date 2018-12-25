package oras

import (
	"context"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Pull pull files from the remote
func Pull(ctx context.Context, resolver remotes.Resolver, ref string) (map[string][]byte, error) {
	if resolver == nil {
		return nil, ErrResolverUndefined
	}

	_, desc, err := resolver.Resolve(ctx, ref)
	if err != nil {
		return nil, err
	}
	fetcher, err := resolver.Fetcher(ctx, ref)
	if err != nil {
		return nil, err
	}

	var layers []ocispec.Descriptor
	picker := images.HandlerFunc(func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		if desc.MediaType == ocispec.MediaTypeImageLayer {
			layers = append(layers, desc)
		}
		return nil, nil
	})
	store := NewMemoryStore()
	handlers := images.Handlers(store.FetchHandler(fetcher), picker, images.ChildrenHandler(store))
	if err := images.Dispatch(ctx, handlers, desc); err != nil {
		return nil, err
	}

	res := make(map[string][]byte)
	for _, layer := range layers {
		if content, ok := store.Get(layer); ok {
			if title, ok := layer.Annotations[ocispec.AnnotationTitle]; ok && len(title) > 0 {
				res[title] = content
			} else {
				res[layer.Digest.Encoded()] = content
			}
		}
	}

	return res, nil
}
