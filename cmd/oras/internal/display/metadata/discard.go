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

package metadata

import (
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type discard struct{}

// NewDiscardHandler creates a new handler that discards output for all events.
func NewDiscardHandler() discard {
	return discard{}
}

// OnFetched implements ManifestFetchHandler.
func (discard) OnFetched(string, ocispec.Descriptor, []byte) error {
	return nil
}

// OnTagged implements ManifestIndexCreateHandler.
func (discard) OnTagged(ocispec.Descriptor, string) error {
	return nil
}

// OnCompleted implements ManifestIndexCreateHandler.
func (discard) OnCompleted(ocispec.Descriptor) error {
	return nil
}

// OnIndexPacked implements ManifestIndexCreateHandler.
func (discard) OnIndexPacked(ocispec.Descriptor) error {
	return nil
}

// OnIndexPushed implements ManifestIndexCreateHandler.
func (discard) OnIndexPushed(string) error {
	return nil
}

// OnManifestRemoved implements ManifestIndexUpdateHandler.
func (discard) OnManifestRemoved(digest.Digest) error {
	return nil
}

// OnManifestAdded implements ManifestIndexUpdateHandler.
func (discard) OnManifestAdded(string, digest.Digest) error {
	return nil
}

// OnIndexMerged implements ManifestIndexUpdateHandler.
func (discard) OnIndexMerged(string, digest.Digest) error {
	return nil
}
