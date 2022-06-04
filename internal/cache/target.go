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
	"oras.land/oras-go/v2/content/oci"
)

// Cache target struct.
type target struct {
	oras.Target
	cache oras.Target
}

// New generates a new target storage with caching.
func New(base oras.Target, root string) (oras.Target, error) {
	cache, err := oci.New(root)
	if err != nil {
		return nil, err
	}
	return &target{
		Target: base,
		cache:  cache,
	}, nil
}

// Push pushes the descriptor into target storage with caching.
func (s *target) Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) error {
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
