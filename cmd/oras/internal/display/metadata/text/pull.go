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
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/model"
	"oras.land/oras/cmd/oras/internal/option"
)

// PullHandler handles text metadata output for pull events.
type PullHandler struct {
	out io.Writer
}

// OnCompleted implements metadata.PullHandler.
func (p *PullHandler) OnCompleted(opts *option.Target, desc ocispec.Descriptor, layerSkipped bool, _ []model.File) error {
	if layerSkipped {
		_, _ = fmt.Fprintf(p.out, "Skipped pulling layers without file name in %q\n", ocispec.AnnotationTitle)
		_, _ = fmt.Fprintf(p.out, "Use 'oras copy %s --to-oci-layout <layout-dir>' to pull all layers.\n", opts.RawReference)
	} else {
		_, _ = fmt.Fprintln(p.out, "Pulled", opts.AnnotatedReference())
		_, _ = fmt.Fprintln(p.out, "Digest:", desc.Digest)
	}
	return nil
}

func (p *PullHandler) OnFilePulled(name string, outputDir string, desc ocispec.Descriptor, descPath string) {
}

// NewPullHandler returns a new handler for Pull events.
func NewPullHandler(out io.Writer) metadata.PullHandler {
	return &PullHandler{out: out}
}
