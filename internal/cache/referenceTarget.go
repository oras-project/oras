/*
Copyright The ORAS Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cache

import (
	"context"
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry"
)

// Cache referenceTarget struct.
type referenceTarget struct {
	*target
	registry.ReferenceFetcher
}

type iReferenceTarget interface {
	oras.Target
	registry.ReferenceFetcher
}

// NewReferenceTarget generates a target with ReferenceFetch and caching.
func NewReferenceTarget(origin iReferenceTarget, cache content.Storage) *referenceTarget {
	target := &target{
		Target: origin,
		cache:  cache,
	}

	return &referenceTarget{
		target:           target,
		ReferenceFetcher: origin,
	}
}

// FetchReference fetches the content identified by the reference from the
// remote and cache the fetched content.
// Cached content will only be read via Fetch, FetchReference will always fetch
// From origin.
func (r *referenceTarget) FetchReference(ctx context.Context, reference string) (ocispec.Descriptor, io.ReadCloser, error) {
	if r.ReferenceFetcher == nil {
		return ocispec.Descriptor{}, nil, errdef.ErrUnsupported
	}
	target, rc, err := r.ReferenceFetcher.FetchReference(ctx, reference)
	if err != nil {
		return ocispec.Descriptor{}, nil, err
	}

	// skip caching if the content already exists in cache
	exists, err := r.cache.Exists(ctx, target)
	if err != nil {
		return ocispec.Descriptor{}, nil, err
	}
	if exists {
		// no need to do tee'd push
		return target, rc, nil
	}

	// Fetch from origin with caching
	return target, piped(ctx, rc, target, r.cache), nil
}
