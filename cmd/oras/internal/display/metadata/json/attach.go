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
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/model"
	"oras.land/oras/cmd/oras/internal/option"
)

// AttachHandler handles metadata events for attach command.
type AttachHandler struct{}

// NewAttachHandler returns a new hander for attach metadata events.
func NewAttachHandler() metadata.AttachHandler {
	return AttachHandler{}
}

// OnCompleted is called when the attach operation is completed.
func (AttachHandler) OnCompleted(opts *option.Target, root, subject ocispec.Descriptor) error {
	return printJSON(model.NewPush(root, opts.Path))
}
