package main

import (
	"context"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/content/local"
	"github.com/containerd/containerd/errdefs"
	orascontent "github.com/deislabs/oras/pkg/content"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
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
	var wOpts content.WriterOpts
	for _, opt := range opts {
		if err := opt(&wOpts); err != nil {
			return nil, err
		}
	}
	switch wOpts.Desc.MediaType {
	case ocispec.MediaTypeImageManifest, ocispec.MediaTypeImageIndex:
		return s.cache.Writer(ctx, opts...)
	}

	cacheWriter, err := s.cache.Writer(ctx, opts...)
	if err != nil {
		if !errdefs.IsAlreadyExists(err) {
			return nil, err
		}
		if err := s.applyCache(ctx, wOpts.Desc, opts...); err != nil {
			return nil, err
		}
		return nil, errdefs.ErrAlreadyExists
	}

	_ = cacheWriter
	panic("copy to base on commit not implemented")
}

// applyCache copies the content from cache to the base store
func (s *cachedStore) applyCache(ctx context.Context, desc ocispec.Descriptor, opts ...content.WriterOpt) error {
	cw, err := s.base.Writer(ctx, opts...)
	if err != nil {
		if errdefs.IsAlreadyExists(err) {
			return nil
		}
		return err
	}

	ws, err := cw.Status()
	if err != nil {
		return err
	}

	if ws.Offset != desc.Size {
		ra, err := s.cache.ReaderAt(ctx, desc)
		if err != nil {
			return err
		}
		defer ra.Close()

		if err := content.CopyReaderAt(cw, ra, desc.Size); err != nil {
			return err
		}
	}

	if err := cw.Commit(ctx, desc.Size, desc.Digest); err != nil && !errdefs.IsAlreadyExists(err) {
		return errors.Wrapf(err, "failed commit on ref %q", ws.Ref)
	}

	return nil
}
