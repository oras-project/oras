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
	"sync/atomic"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
)

// helmConfigMediaType is the config media type of a Helm chart artifact.
const helmConfigMediaType = "application/vnd.cncf.helm.config.v1+json"

// PullHandler handles text metadata output for pull events.
type PullHandler struct {
	printer      *output.Printer
	layerSkipped atomic.Bool
	helmChart    atomic.Bool
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
func (ph *PullHandler) OnLayerSkipped(desc ocispec.Descriptor) error {
	ph.layerSkipped.Store(true)
	if desc.MediaType == helmConfigMediaType {
		ph.helmChart.Store(true)
	}
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
		if cmd := ph.helmPullCommand(); cmd != "" {
			_ = ph.printer.Printf("Use '%s' to pull the Helm chart.\n", cmd)
		} else {
			_ = ph.printer.Printf("Use 'oras copy %s --to-oci-layout <layout-dir>' to pull all layers.\n", ph.target.RawReference)
		}
	} else {
		_ = ph.printer.Println("Pulled", ph.target.GetDisplayReference())
		_ = ph.printer.Println("Digest:", ph.root.Digest)
	}
	return nil
}

// helmPullCommand returns the helm command that pulls the target, or an empty
// string if the target is not a Helm chart in a remote registry.
func (ph *PullHandler) helmPullCommand() string {
	if !ph.helmChart.Load() || ph.target.Type != option.TargetTypeRemote {
		return ""
	}
	if _, err := digest.Parse(ph.target.Reference); err == nil {
		return fmt.Sprintf("helm pull oci://%s@%s", ph.target.Path, ph.target.Reference)
	}
	return fmt.Sprintf("helm pull oci://%s --version %s", ph.target.Path, ph.target.Reference)
}
