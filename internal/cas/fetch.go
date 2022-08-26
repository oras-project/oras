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

package cas

import (
	"context"
	"encoding/json"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras/internal/cache"
)

// FetchDescriptor fetches a minimal descriptor of reference from target.
// If platform flag not empty, will fetch the specified platform.
func FetchDescriptor(ctx context.Context, target oras.ReadOnlyTarget, reference string, p *ocispec.Platform) ([]byte, error) {
	desc, err := oras.Resolve(ctx, target, reference, oras.ResolveOptions{TargetPlatform: p})
	if err != nil {
		return nil, err
	}
	return json.Marshal(ocispec.Descriptor{
		MediaType: desc.MediaType,
		Digest:    desc.Digest,
		Size:      desc.Size,
	})
}

// FetchManifest fetches the manifest content of reference from target.
// If platform flag not empty, will fetch the specified platform.
func FetchManifest(ctx context.Context, target oras.ReadOnlyTarget, reference string, p *ocispec.Platform) ([]byte, error) {
	// TODO: improve implementation once oras-go#102 is resolved
	if p == nil {
		if rf, ok := target.(registry.ReferenceFetcher); ok {
			desc, rc, err := rf.FetchReference(ctx, reference)
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return content.ReadAll(rc, desc)
		}
	}
	target = cache.New(target, memory.New())
	desc, err := oras.Resolve(ctx, target, reference, oras.ResolveOptions{
		TargetPlatform: p,
	})
	if err != nil {
		return nil, err
	}
	rc, err := target.Fetch(ctx, desc)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return content.ReadAll(rc, desc)
}

// FetchBlob fetches the blob content of reference from blob store.
func FetchBlob(ctx context.Context, blob oras.Target, reference string) ([]byte, error) {
	rf := blob.(registry.ReferenceFetcher)
	desc, rc, err := rf.FetchReference(ctx, reference)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return content.ReadAll(rc, desc)
}

// BlobTarget returns a ORAS Target with a no-op Tag method wrapping the
// provided blob store b.
func BlobTarget(b registry.BlobStore) oras.Target {
	return blobTarget{b}
}

type blobTarget struct {
	registry.BlobStore
}

func (blobTarget) Tag(ctx context.Context, desc ocispec.Descriptor, reference string) error {
	return nil
}
