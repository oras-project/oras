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

// ManifestFetchHandler handles raw output for manifest fetch events.
type ManifestFetchHandler interface {
	// OnFetched is called after the manifest content is fetched.
	OnContentFetched(outputPath string, content []byte) error
	// OnDescriptorFetched is called after the manifest descriptor is
	// fetched.
	OnDescriptorFetched(desc ocispec.Descriptor) error
}