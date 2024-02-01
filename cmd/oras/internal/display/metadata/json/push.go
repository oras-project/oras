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

type PushHandler struct {
	path string
}

func NewPushHandler() metadata.PushHandler {
	return &PushHandler{}
}

func (ph *PushHandler) OnCopied(opts *option.Target) error {
	ph.path = opts.Path
	return nil
}

func (ph *PushHandler) OnTagged(reference string) error {
	return nil
}

func (ph *PushHandler) OnCompleted(root ocispec.Descriptor) error {
	return printJSON(model.NewPush(root, ph.path))
}
