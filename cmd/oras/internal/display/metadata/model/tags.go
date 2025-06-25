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

// Tags contains metadata formatted by oras repo tags.
type Tags struct {
	Tags []string `json:"tags"`
}

// NewTags creates a new Tags model.
func NewTags() *Tags {
	return &Tags{
		Tags: []string{},
	}
}

// AddTag adds a tag to the metadata.
func (t *Tags) AddTag(tag string) {
	t.Tags = append(t.Tags, tag)
}
