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
	"oras.land/oras/cmd/oras/internal/display/metadata/model"
	"oras.land/oras/cmd/oras/internal/option"
)

// PullHandler handles text metadata output for pull events.
type PullHandler struct{}

// OnCompleted implements metadata.PullHandler.
func (p *PullHandler) OnCompleted(opts *option.Target, desc ocispec.Descriptor, layerSkipped bool, _ []model.File) error {
	if layerSkipped {
		_, _ = fmt.Printf("Skipped pulling layers without file name in %q\n", ocispec.AnnotationTitle)
		_, _ = fmt.Printf("Use 'oras copy %s --to-oci-layout <layout-dir>' to pull all layers.\n", opts.RawReference)
	} else {
		_, _ = fmt.Println("Pulled", opts.AnnotatedReference())
		_, _ = fmt.Println("Digest:", desc.Digest)
	}
	return nil
}

// NewPullHandler returns a new handler for Pull events.
func NewPullHandler() metadata.PullHandler {
	return &PullHandler{}
}
