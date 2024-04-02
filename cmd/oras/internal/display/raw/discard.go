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

package raw

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type discard struct{}

// OnContentFetched implements ManifestFetchHandler.
func (discard) OnContentFetched(string, []byte) error { return nil }

// OnDescriptorFetched implements ManifestFetchHandler.
func (discard) OnDescriptorFetched(desc ocispec.Descriptor) error { return nil }

// NewManifestFetchHandler creates a new handler.
func NewDiscardHandler() ManifestFetchHandler {
	return discard{}
}
