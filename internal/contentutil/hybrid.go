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

type hybrid struct {
	oras.ReadOnlyTarget
	toUnion []oras.Target
}

// MultiReadOnlyTarget generates a new hybrid storage.
func MultiReadOnlyTarget(provider oras.ReadOnlyTarget, toUnion ...oras.Target) oras.ReadOnlyTarget {
	return &hybrid{
		ReadOnlyTarget: provider,
		toUnion:        toUnion,
	}
}

// Fetch fetches the content from combined targets first, then from the provider.
func (h *hybrid) Fetch(ctx context.Context, target ocispec.Descriptor) (io.ReadCloser, error) {
	for _, c := range h.toUnion {
		rc, err := c.Fetch(ctx, target)
		if err == nil {
			return rc, nil
		}
	}
	return h.ReadOnlyTarget.Fetch(ctx, target)
}

// Resolve resolves the content from cache first, then from the provider.
func (h *hybrid) Resolve(ctx context.Context, ref string) (ocispec.Descriptor, error) {
	for _, c := range h.toUnion {
		desc, err := c.Resolve(ctx, ref)
		if err == nil {
			return desc, nil
		}
	}
	return h.ReadOnlyTarget.Resolve(ctx, ref)
}
