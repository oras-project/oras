package main

import (
	"context"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/content/local"
	"github.com/containerd/containerd/errdefs"
	orascontent "github.com/deislabs/oras/pkg/content"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type cachedStore struct {
	base  content.Ingester
	cache content.Store
}

// newStoreWithCache creates a store with a consistent cache layer.
func newStoreWithCache(base content.Ingester, cacheRoot string) (orascontent.ProvideIngester, error) {
	cache, err := local.NewStore(cacheRoot)
	if err != nil {
		return nil, err
	}
	return &cachedStore{
		base:  base,
		cache: cache,
	}, nil
}

// ReaderAt reads the cache if available, and then check the base store.
func (s *cachedStore) ReaderAt(ctx context.Context, desc ocispec.Descriptor) (content.ReaderAt, error) {
	return s.cache.ReaderAt(ctx, desc)
}

// Writer writes to the cache if not exists, and then copy the cache to the base store.
func (s *cachedStore) Writer(ctx context.Context, opts ...content.WriterOpt) (content.Writer, error) {
	cacheWriter, err := s.cache.Writer(ctx, opts...)
	if err != nil {
		if !errdefs.IsAlreadyExists(err) {
			return nil, err
		}
		if err := s.syncWrite(ctx, opts...); err != nil {
			return nil, err
		}
		return nil, errdefs.ErrAlreadyExists
	}

	_ = cacheWriter
	panic("copy to base on commit not implemented")
}

func (s *cachedStore) syncWrite(ctx context.Context, opts ...content.WriterOpt) error {
	panic("not implemented")
}
