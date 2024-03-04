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
	"oras.land/oras/cmd/oras/internal/option"
)

// PushHandler handles metadata output for push events.
type PushHandler interface {
	OnCopied(opts *option.Target) error
	OnCompleted(root ocispec.Descriptor) error
}

// AttachHandler handles metadata output for attach events.
type AttachHandler interface {
	OnCompleted(opts *option.Target, root, subject ocispec.Descriptor) error
}
