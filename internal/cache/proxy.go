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
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry"
)

type closer func() error

func (fn closer) Close() error {
	return fn()
}

// Cache proxy struct.
type proxy struct {
	oras.Target
	registry.ReferenceFetcher
	cache content.Storage
}

// New generates a new target storage with caching.
func New(source oras.Target, cache content.Storage) oras.Target {
	return &proxy{
		Target: source,
		cache:  cache,
	}
}

// Fetch fetches the content identified by the descriptor.
func (p *proxy) Fetch(ctx context.Context, target ocispec.Descriptor) (io.ReadCloser, error) {
	rc, err := p.cache.Fetch(ctx, target)
	if err == nil {
		// Fetch from cache
		return rc, nil
	}

	if rc, err = p.Target.Fetch(ctx, target); err != nil {
		return nil, err
	}

	// Fetch from origin with caching
	return withCaching(ctx, rc, target, p.cache), nil
}

// FetchReference fetches the content identified by the reference from the
// remote and cache the fetched content.
// Cached content will only be read via Fetch, FetchReference will always fetch
// From origin.
func (p *proxy) FetchReference(ctx context.Context, reference string) (ocispec.Descriptor, io.ReadCloser, error) {
	target, rc, err := p.ReferenceFetcher.FetchReference(ctx, reference)
	if err != nil {
		return ocispec.Descriptor{}, nil, err
	}

	// skip caching if the content already exists in cache
	exists, err := p.cache.Exists(ctx, target)
	if err != nil {
		return ocispec.Descriptor{}, nil, err
	}
	if exists {
		// no need to do tee'd push
		return target, rc, nil
	}

	// Fetch from origin with caching
	return target, withCaching(ctx, rc, target, p.cache), nil
}

func withCaching(ctx context.Context, in io.ReadCloser, target ocispec.Descriptor, cache content.Storage) io.ReadCloser {
	pr, pw := io.Pipe()
	var wg sync.WaitGroup
	wg.Add(1)

	var pushErr error
	go func() {
		defer wg.Done()
		pushErr = cache.Push(ctx, target, pr)
	}()

	return struct {
		io.Reader
		io.Closer
	}{
		Reader: io.TeeReader(in, pw),
		Closer: closer(func() error {
			rcErr := in.Close()
			if err := pw.Close(); err != nil {
				return err
			}
			wg.Wait()
			if pushErr != nil {
				return pushErr
			}
			return rcErr
		}),
	}
}

// Exists returns true if the described content exists.
func (p *proxy) Exists(ctx context.Context, desc ocispec.Descriptor) (bool, error) {
	exists, err := p.cache.Exists(ctx, desc)
	if err == nil && exists {
		return true, nil
	}
	return p.Target.Exists(ctx, desc)
}
