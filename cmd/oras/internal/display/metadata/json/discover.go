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

package json

import (
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/model"
)

// DiscoverHandler handles json metadata output for discover events.
type DiscoverHandler struct {
	path string
}

// OnDiscovered implements metadata.DiscoverHandler.
func (d DiscoverHandler) OnDiscovered(refs []v1.Descriptor) error {
	return PrintJSON(model.NewDiscover(d.path, refs))
}

// NewDiscoverHandler creates a new handler for discover events.
func NewDiscoverHandler(path string) metadata.DiscoverHandler {
	return DiscoverHandler{
		path: path,
	}
}
