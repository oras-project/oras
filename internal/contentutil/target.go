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

package contentutil

import (
	"context"
	"errors"
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/errdef"
)

type multiReadOnlyTarget struct {
	targets []oras.ReadOnlyTarget
}

// MultiReadOnlyTarget generates a new hybrid storage.
func MultiReadOnlyTarget(targets ...oras.ReadOnlyTarget) oras.ReadOnlyTarget {
	return &multiReadOnlyTarget{
		targets: targets,
	}
}

// Fetch fetches the content from combined targets first, then from the provider.
func (m *multiReadOnlyTarget) Fetch(ctx context.Context, target ocispec.Descriptor) (io.ReadCloser, error) {
	lastErr := errdef.ErrNotFound
	for _, c := range m.targets {
		rc, err := c.Fetch(ctx, target)
		if err != nil {
			if errors.Is(err, errdef.ErrNotFound) {
				lastErr = err
				continue
			}
			return nil, err
		}
		return rc, nil
	}
	return nil, lastErr
}

// Exists returns true if the described content exists.
func (m *multiReadOnlyTarget) Exists(ctx context.Context, target ocispec.Descriptor) (bool, error) {
	lastErr := errdef.ErrNotFound
	for _, c := range m.targets {
		exists, err := c.Exists(ctx, target)
		if err != nil {
			if errors.Is(err, errdef.ErrNotFound) {
				lastErr = err
				continue
			}
			return false, err
		}
		return exists, nil
	}
	return false, lastErr
}

// Resolve resolves the content from cache first, then from the provider.
func (m *multiReadOnlyTarget) Resolve(ctx context.Context, ref string) (ocispec.Descriptor, error) {
	lastErr := errdef.ErrNotFound
	for _, c := range m.targets {
		desc, err := c.Resolve(ctx, ref)
		if err != nil {
			if errors.Is(err, errdef.ErrNotFound) {
				lastErr = err
				continue
			}
			return ocispec.Descriptor{}, err
		}
		return desc, nil
	}
	return ocispec.Descriptor{}, lastErr
}
