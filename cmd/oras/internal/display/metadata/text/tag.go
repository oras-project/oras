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
)

// TagHandler handles text metadata output for tag events.
type TagHandler struct {
	out io.Writer
}

// OnTagged implements metadata.TextTagHandler.
func (h *TagHandler) OnTagged(_ ocispec.Descriptor, tag string) error {
	_, err := fmt.Fprintln(h.out, "Tagged", tag)
	return err
}

// NewTagHandler returns a new handler for tag events.
func NewTagHandler(out io.Writer) metadata.TagHandler {
	return &TagHandler{
		out: out,
	}
}
