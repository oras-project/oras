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

package model

// Repositories contains metadata formatted by oras repo ls.
type Repositories struct {
	Registry     string   `json:"registry"`
	Repositories []string `json:"repositories"`
}

// NewRepositories creates a new Repositories model.
func NewRepositories(registry string) *Repositories {
	return &Repositories{
		Registry:     registry,
		Repositories: []string{},
	}
}

// AddRepository adds a repository to the metadata.
func (r *Repositories) AddRepository(repo string) {
	r.Repositories = append(r.Repositories, repo)
}
