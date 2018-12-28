package oras

import (
	"context"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/remotes"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Pull pull files from the remote
func Pull(ctx context.Context, resolver remotes.Resolver, ref string, allowedMediaTypes ...string) (map[string]Blob, error) {
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

	var blobs []ocispec.Descriptor
	picker := images.HandlerFunc(func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		if isAllowedMediaType(desc.MediaType, allowedMediaTypes...) {
			blobs = append(blobs, desc)
			return nil, nil
		}
		return nil, nil
	})
	store := NewMemoryStore()
	handlers := images.Handlers(
		filterHandler(allowedMediaTypes...),
		store.FetchHandler(fetcher),
		picker,
		images.ChildrenHandler(store),
	)
	if err := images.Dispatch(ctx, handlers, desc); err != nil {
		return nil, err
	}

	res := make(map[string]Blob)
	for _, blob := range blobs {
		if content, ok := store.Get(blob); ok {
			if name, ok := blob.Annotations[ocispec.AnnotationTitle]; ok && len(name) > 0 {
				res[name] = Blob{
					MediaType: blob.MediaType,
					Content:   content,
				}
			}
		}
	}

	return res, nil
}

func filterHandler(allowedMediaTypes ...string) images.HandlerFunc {
	return func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		switch {
		case isAllowedMediaType(desc.MediaType, ocispec.MediaTypeImageManifest, ocispec.MediaTypeImageIndex):
			return nil, nil
		case isAllowedMediaType(desc.MediaType, allowedMediaTypes...):
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

func isAllowedMediaType(mediaType string, allowedMediaTypes ...string) bool {
	if len(allowedMediaTypes) == 0 {
		return true
	}
	for _, allowedMediaType := range allowedMediaTypes {
		if mediaType == allowedMediaType {
			return true
		}
	}
	return false
}
