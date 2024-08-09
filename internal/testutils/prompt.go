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

package testutils

import (
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
)

// PromptDiscarder mocks trackable GraphTarget with discarded prompt.
type PromptDiscarder struct {
	oras.GraphTarget
	io.Closer
}

// Prompt discards the prompt.
func (p *PromptDiscarder) Prompt(ocispec.Descriptor, string) error {
	return nil
}

// ErrorPrompt mocks an errored prompt.
type ErrorPrompt struct {
	oras.GraphTarget
	io.Closer
	wanted error
}

// NewErrorPrompt creates an error prompt.
func NewErrorPrompt(err error) *ErrorPrompt {
	return &ErrorPrompt{
		wanted: err,
	}
}

// Prompt mocks an errored prompt.
func (e *ErrorPrompt) Prompt(ocispec.Descriptor, string) error {
	return e.wanted
}
