package oras

import (
	"context"
	"testing"

	"github.com/containerd/containerd/remotes"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	artifactspec "github.com/oras-project/artifacts-spec/specs-go/v1"
)

type (
	testResolverWithDiscover struct {
		remotes.Resolver
	}
	testResolverWithoutDiscover struct {
		remotes.Resolver
	}
)

func (testResolverWithDiscover) Resolve(ctx context.Context, ref string) (string, ocispec.Descriptor, error) {
	return "", ocispec.Descriptor{MediaType: "test-media-type"}, nil
}

func (testResolverWithDiscover) Discover(ctx context.Context, desc ocispec.Descriptor, artifactType string) ([]artifactspec.Descriptor, error) {
	return []artifactspec.Descriptor{
		{
			MediaType:    desc.MediaType,
			ArtifactType: artifactType,
		}}, nil
}

func TestDiscoverInterfaceOverride(t *testing.T) {
	d, a, err := Discover(context.Background(), &testResolverWithDiscover{}, "testref", "testartifacttype")
	if err != nil {
		t.Error(err)
	}

	if d.MediaType != "test-media-type" {
		t.FailNow()
	}

	if a[0].ArtifactType != "testartifacttype" {
		t.FailNow()
	}

	if a[0].MediaType != "test-media-type" {
		t.FailNow()
	}
}

func TestDiscoverNotImplemented(t *testing.T) {
	_, _, err := Discover(context.Background(), &testResolverWithoutDiscover{}, "testref", "testartifacttype")
	if err == nil {
		t.FailNow()
	}

	if err.Error() != "not implemented" {
		t.FailNow()
	}
}
