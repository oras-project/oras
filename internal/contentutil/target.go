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
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
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
	var rc io.ReadCloser
	var err error

	for _, c := range m.targets {
		rc, err = c.Fetch(ctx, target)
		if err == nil {
			break
		}
	}
	return rc, err
}

// Exists returns true if the described content exists.
func (m *multiReadOnlyTarget) Exists(ctx context.Context, target ocispec.Descriptor) (bool, error) {
	var exists bool
	var err error

	for _, c := range m.targets {
		exists, err = c.Exists(ctx, target)
		if err == nil {
			break
		}
	}
	return exists, err
}

// Resolve resolves the content from cache first, then from the provider.
func (m *multiReadOnlyTarget) Resolve(ctx context.Context, ref string) (ocispec.Descriptor, error) {
	var desc ocispec.Descriptor
	var err error

	for _, c := range m.targets {
		desc, err = c.Resolve(ctx, ref)
		if err == nil {
			break
		}
	}
	return desc, err
}
