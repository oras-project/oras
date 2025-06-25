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

	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/model"
	"oras.land/oras/cmd/oras/internal/output"
)

// repoTagsHandler handles JSON metadata output for repo tags command.
type repoTagsHandler struct {
	out   io.Writer
	model *model.Tags
}

// NewRepoTagsHandler creates a new handler for repo tags events.
func NewRepoTagsHandler(out io.Writer) metadata.RepoTagsHandler {
	return &repoTagsHandler{
		out:   out,
		model: model.NewTags(),
	}
}

// OnTagListed implements metadata.TagsHandler.
func (h *repoTagsHandler) OnTagListed(tag string) error {
	h.model.AddTag(tag)
	return nil
}

// Render implements metadata.TagsHandler.
func (h *repoTagsHandler) Render() error {
	return output.PrintPrettyJSON(h.out, h.model)
}
