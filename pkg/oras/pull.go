package oras

import (
	"context"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/log"
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
	handlers := images.Handlers(filterHandler(), store.FetchHandler(fetcher), picker, images.ChildrenHandler(store))
	if err := images.Dispatch(ctx, handlers, desc); err != nil {
		return nil, err
	}

	res := make(map[string][]byte)
	for _, layer := range layers {
		if content, ok := store.Get(layer); ok {
			if name, ok := layer.Annotations[ocispec.AnnotationTitle]; ok && len(name) > 0 {
				res[name] = content
			}
		}
	}

	return res, nil
}

func filterHandler() images.HandlerFunc {
	return func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		switch desc.MediaType {
		case ocispec.MediaTypeImageManifest, ocispec.MediaTypeImageIndex:
			return nil, nil
		case ocispec.MediaTypeImageLayer:
			if name, ok := desc.Annotations[ocispec.AnnotationTitle]; ok && len(name) > 0 {
				return nil, nil
			}
			log.G(ctx).Warnf("layer_no_name: %v", desc.Digest)
		default:
			log.G(ctx).Warnf("unknown_type: %v", desc.MediaType)
		}
		return nil, images.ErrStopHandler
	}
}
