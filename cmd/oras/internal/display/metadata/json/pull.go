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
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/model"
	"oras.land/oras/cmd/oras/internal/option"
)

// PullHandler handles JSON metadata output for pull events.
type PullHandler struct {
	path   string
	out    io.Writer
	pulled model.Pulled
}

// NewPullHandler returns a new handler for Pull events.
func NewPullHandler(path string, out io.Writer) metadata.PullHandler {
	return &PullHandler{
		path: path,
		out:  out,
	}
}

// OnFilePulled implements metadata.PullHandler.
func (ph *PullHandler) OnFilePulled(name string, outputDir string, desc ocispec.Descriptor, descPath string) {
	ph.pulled.Add(name, outputDir, desc, descPath)
}

// OnCompleted implements metadata.PullHandler.
func (ph *PullHandler) OnCompleted(opts *option.Target, desc ocispec.Descriptor, _ bool) error {
	return printJSON(model.NewPull(ph.path+"@"+desc.Digest.String(), ph.pulled.Files))
}
