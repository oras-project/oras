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

package content

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ManifestFetchHandler handles raw output for manifest fetch events.
type ManifestFetchHandler interface {
	// OnContentFetched is called after the manifest content is fetched.
	OnContentFetched(desc ocispec.Descriptor, content []byte) error
}

// ManifestIndexCreateHandler handles raw output for manifest index create events.
type ManifestIndexCreateHandler interface {
	// OnContentCreated is called after the index content is created.
	OnContentCreated(content []byte) error
}

// ManifestIndexUpdateHandler handles raw output for manifest index update events.
type ManifestIndexUpdateHandler ManifestIndexCreateHandler
