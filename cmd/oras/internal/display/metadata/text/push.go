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

package text

import (
	"fmt"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/option"
)

type PushHandler struct{}

func NewPushHandler() metadata.PushHandler {
	return PushHandler{}
}

func (PushHandler) OnCopied(opts *option.Target) error {
	_, err := fmt.Println("Pushed", opts.AnnotatedReference())
	return err
}

func (PushHandler) OnTagged(reference string) error {
	panic("not implemented")
}

func (PushHandler) OnCompleted(root ocispec.Descriptor) error {
	_, err := fmt.Println("Digest:", root.Digest)
	return err
}
