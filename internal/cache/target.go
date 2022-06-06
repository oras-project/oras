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

// New generates a new target storage with caching.
func New(source oras.Target, cache content.Storage) oras.Target {
	return &target{
		Target: source,
		cache:  cache,
	}
}

// Fetch fetches the content identified by the descriptor.
func (p *target) Fetch(ctx context.Context, target ocispec.Descriptor) (io.ReadCloser, error) {
	rc, err := p.cache.Fetch(ctx, target)
	if err == nil {
		return rc, nil
	}

	rc, err = p.Target.Fetch(ctx, target)
	if err != nil {
		return nil, err
	}
	pr, pw := io.Pipe()
	var wg sync.WaitGroup
	wg.Add(1)
	var pushErr error
	go func() {
		defer wg.Done()
		pushErr = p.cache.Push(ctx, target, pr)
	}()
	c := closer(func() error {
		rcErr := rc.Close()
		if err := pw.Close(); err != nil {
			return err
		}
		wg.Wait()
		if pushErr != nil {
			return pushErr
		}
		return rcErr
	})

	return struct {
		io.Reader
		io.Closer
	}{
		Reader: io.TeeReader(rc, pw),
		Closer: c,
	}, nil
}

// Exists returns true if the described content exists.
func (p *target) Exists(ctx context.Context, desc ocispec.Descriptor) (bool, error) {
	exists, err := p.cache.Exists(ctx, desc)
	if err == nil && exists {
		return true, nil
	}
	return p.Target.Exists(ctx, desc)
}
