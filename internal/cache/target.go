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
)

type closer func() error

func (fn closer) Close() error {
	return fn()
}

// Cache target struct.
type target struct {
	oras.Target
	cache content.Storage
}

// NewTarget generates a new target with caching.
func NewTarget(origin oras.Target, cache content.Storage) *target {
	return &target{
		Target: origin,
		cache:  cache,
	}
}

// Fetch fetches the content identified by the descriptor.
func (t *target) Fetch(ctx context.Context, target ocispec.Descriptor) (io.ReadCloser, error) {
	rc, err := t.cache.Fetch(ctx, target)
	if err == nil {
		// Fetch from cache
		return rc, nil
	}

	if rc, err = t.Target.Fetch(ctx, target); err != nil {
		return nil, err
	}

	// Fetch from origin with caching
	return piped(ctx, rc, target, t.cache), nil
}

func piped(ctx context.Context, in io.ReadCloser, target ocispec.Descriptor, cache content.Storage) io.ReadCloser {
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
func (t *target) Exists(ctx context.Context, desc ocispec.Descriptor) (bool, error) {
	exists, err := t.cache.Exists(ctx, desc)
	if err == nil && exists {
		return true, nil
	}
	return t.Target.Exists(ctx, desc)
}
