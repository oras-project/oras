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
	"sync/atomic"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
)

// PullHandler handles text metadata output for pull events.
type PullHandler struct {
	printer      *output.Printer
	layerSkipped atomic.Bool
	target       *option.Target
	root         ocispec.Descriptor
}

// NewPullHandler returns a new handler for Pull events.
func NewPullHandler(printer *output.Printer) metadata.PullHandler {
	return &PullHandler{
		printer: printer,
	}
}

func (ph *PullHandler) OnFilePulled(_ string, _ string, _ ocispec.Descriptor, _ string) error {
	return nil
}

// OnLayerSkipped implements metadata.PullHandler.
func (ph *PullHandler) OnLayerSkipped(ocispec.Descriptor) error {
	ph.layerSkipped.Store(true)
	return nil
}

// OnPulled implements metadata.PullHandler.
func (ph *PullHandler) OnPulled(target *option.Target, desc ocispec.Descriptor) {
	ph.target = target
	ph.root = desc
}

// Render implements metadata.PullHandler.
func (ph *PullHandler) Render() error {
	if ph.layerSkipped.Load() {
		_ = ph.printer.Printf("Skipped pulling layers without file name in %q\n", ocispec.AnnotationTitle)
		_ = ph.printer.Printf("Use 'oras copy %s --to-oci-layout <layout-dir>' to pull all layers.\n", ph.target.RawReference)
	} else {
		_ = ph.printer.Println("Pulled", ph.target.AnnotatedReference())
		_ = ph.printer.Println("Digest:", ph.root.Digest)
	}
	return nil
}
