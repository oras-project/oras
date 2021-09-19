package oras

import (
	"context"
	"errors"

	"github.com/containerd/containerd/remotes"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	artifactspec "github.com/oras-project/artifacts-spec/specs-go/v1"
)

// Discover discovers artifacts referencing the specified artifact
func Discover(ctx context.Context, resolver remotes.Resolver, ref, artifactType string) (ocispec.Descriptor, []artifactspec.Descriptor, error) {
	discoverer, ok := resolver.(interface {
		Discover(ctx context.Context, desc ocispec.Descriptor, artifactType string) ([]artifactspec.Descriptor, error)
	})

	if !ok {
		return ocispec.Descriptor{}, nil, errors.New("not implemented")
	}

	_, desc, err := resolver.Resolve(ctx, ref)
	if err != nil {
		return ocispec.Descriptor{}, nil, err
	}

	artifacts, err := discoverer.Discover(ctx, desc, artifactType)
	if err != nil {
		return ocispec.Descriptor{}, nil, err
	}

	return desc, artifacts, err
}
