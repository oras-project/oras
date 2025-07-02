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

package template

import (
	"io"

	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/model"
	"oras.land/oras/cmd/oras/internal/output"
)

// repoListHandler handles template metadata output for repo ls command.
type repoListHandler struct {
	out      io.Writer
	model    *model.Repositories
	template string
}

// NewRepoListHandler creates a new template handler for repo ls command.
func NewRepoListHandler(out io.Writer, tmpl string, registry string) metadata.RepoListHandler {
	return &repoListHandler{
		out:      out,
		model:    model.NewRepositories(registry),
		template: tmpl,
	}
}

// OnRepositoryListed implements metadata.RepoListHandler.
func (h *repoListHandler) OnRepositoryListed(repo string) error {
	// For Go-template format, show the full repository name
	h.model.AddRepository(repo)
	return nil
}

// Render implements metadata.RepoListHandler.
func (h *repoListHandler) Render() error {
	return output.ParseAndWrite(h.out, h.model, h.template)
}
