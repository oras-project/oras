package resolver

import (
	"context"
	"io"
	"time"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/remotes"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

// dummyResolver is a dummy resolver, which resolves nothing.
// It accepts any pushes but ignores them.
type dummyResolver struct{}

var dummyResolverInstance = &dummyResolver{}

// Dummy creates a new dummy resolver
func Dummy() remotes.Resolver {
	return dummyResolverInstance
}

// IsDummy checks if the resolver is dummy
func IsDummy(resolver remotes.Resolver) bool {
	return resolver == dummyResolverInstance
}

func (r *dummyResolver) Resolve(ctx context.Context, ref string) (name string, desc ocispec.Descriptor, err error) {
	return "", ocispec.Descriptor{}, errors.Wrap(errdefs.ErrNotFound, "dummy resolver")
}

func (r *dummyResolver) Fetcher(ctx context.Context, ref string) (remotes.Fetcher, error) {
	return remotes.FetcherFunc(func(ctx context.Context, desc ocispec.Descriptor) (io.ReadCloser, error) {
		return nil, errors.Wrap(errdefs.ErrNotFound, "dummy resolver")
	}), nil
}

// Pusher returns a new pusher for the provided reference
func (r *dummyResolver) Pusher(ctx context.Context, ref string) (remotes.Pusher, error) {
	return remotes.PusherFunc(func(ctx context.Context, desc ocispec.Descriptor) (content.Writer, error) {
		now := time.Now()
		return &dummyWriter{
			actual: digest.Canonical.Digester(),
			status: content.Status{
				Ref:       ref,
				Total:     desc.Size,
				Expected:  desc.Digest,
				StartedAt: now,
				UpdatedAt: now,
			},
		}, nil
	}), nil
}

// Discoverer returns a new discoverer for the provided reference
func (r *dummyResolver) Discoverer(ctx context.Context, ref string) (remotes.Discoverer, error) {
	return remotes.DiscovererFunc(func(ctx context.Context, desc ocispec.Descriptor, artifactType string) ([]remotes.DiscoveredArtifact, error) {
		return nil, errors.Wrap(errdefs.ErrNotFound, "dummy resolver")
	}), nil
}

type dummyWriter struct {
	actual digest.Digester
	status content.Status
}

func (dw *dummyWriter) Write(p []byte) (n int, err error) {
	n, err = dw.actual.Hash().Write(p)
	dw.status.Offset += int64(n)
	dw.status.UpdatedAt = time.Now()
	return
}

func (dw *dummyWriter) Close() error {
	return nil
}

func (dw *dummyWriter) Status() (content.Status, error) {
	return dw.status, nil
}

func (dw *dummyWriter) Digest() digest.Digest {
	return dw.status.Expected
}

func (dw *dummyWriter) Commit(ctx context.Context, size int64, expected digest.Digest, opts ...content.Opt) error {
	if size > 0 && size != dw.status.Offset {
		return errors.Errorf("unexpected size %d, expected %d", dw.status.Offset, size)
	}
	if expected == "" {
		expected = dw.status.Expected
	}

	actual := dw.actual.Digest()
	if actual != expected {
		return errors.Errorf("got digest %s, expected %s", actual, expected)
	}
	return nil
}

func (dw *dummyWriter) Truncate(size int64) error {
	return errors.New("cannot truncate dummy upload")
}
