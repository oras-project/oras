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

package display

import (
	"context"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// PreCopyStatus returns a tracking function for uploading status.
func PreCopyStatus(skip func() bool) func(context.Context, ocispec.Descriptor) error {
	return func(ctx context.Context, desc ocispec.Descriptor) error {
		name, ok := desc.Annotations[ocispec.AnnotationTitle]
		if !ok {
			if skip() {
				return nil
			}
			name = desc.MediaType
		}
		return Print("Uploading", ShortDigest(desc), name)
	}
}

// PreCopyStatus is a tracking function which will be called when uploading
// can be skipped.
func CopySkippedStatus(ctx context.Context, desc ocispec.Descriptor) error {
	return Print("Exists   ", ShortDigest(desc), desc.Annotations[ocispec.AnnotationTitle])
}
