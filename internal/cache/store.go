package cache

import (
	"context"
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
)

type store struct {
	oras.Target
	cache oras.Target
}

func New(base oras.Target, root string) (oras.Target, error) {
	cache, err := oci.New(root)
	if err != nil {
		return nil, err
	}
	return &store{
		Target: base,
		cache:  cache,
	}, nil
}

// Push pushes the descriptor with caching.
func (s *store) Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) error {
	existed, err := s.cache.Exists(ctx, expected)
	if err != nil {
		return err
	}
	if !existed {
		return s.cache.Push(ctx, expected, content)
	}
	rc, err := s.cache.Fetch(ctx, expected)
	if err != nil {
		return err
	}
	return s.Target.Push(ctx, expected, rc)
}
