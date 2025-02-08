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
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type Discard struct{}

// NewDiscardHandler creates a new handler that discards output for all events.
func NewDiscardHandler() Discard {
	return Discard{}
}

// OnFetched implements ManifestFetchHandler.
func (Discard) OnFetched(string, ocispec.Descriptor, []byte) error {
	return nil
}

// OnManifestPushed implements ManifestPushHandler.
func (Discard) OnManifestPushed(ocispec.Descriptor) error {
	return nil
}

// Render implements ManifestPushHandler.
func (Discard) Render() error {
	return nil
}

// OnTagged implements ManifestIndexCreateHandler.
func (Discard) OnTagged(ocispec.Descriptor, string) error {
	return nil
}

// OnIndexCreated implements ManifestIndexCreateHandler.
func (Discard) OnIndexCreated(ocispec.Descriptor) {}
