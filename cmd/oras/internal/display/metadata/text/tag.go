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
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/output"
	"sync"
)

// TagHandler handles text metadata output for tag events.
type TagHandler struct {
	printer   *output.Printer
	printOnce sync.Once
	refPrefix string
}

// NewTagHandler returns a new handler for attach events.
func NewTagHandler(printer *output.Printer, refPrefix string) metadata.TagHandler {
	return &TagHandler{
		printer:   printer,
		refPrefix: refPrefix,
	}
}

// OnTagging is called when the tagging is complete.
func (ah *TagHandler) OnTagging(desc ocispec.Descriptor, _ string) (err error) {
	ah.printOnce.Do(func() {
		ref := ah.refPrefix + "@" + desc.Digest.String()
		err = ah.printer.Println("Tagging", ref)
	})
	return err
}

// OnTagged is called when the tagging is complete.
func (ah *TagHandler) OnTagged(_ ocispec.Descriptor, tag string) error {
	return ah.printer.Println("Tagged", tag)
}
