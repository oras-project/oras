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

package errors

import (
	"fmt"

	"oras.land/oras-go/v2/registry"
)

type output struct {
	description, usage, suggestion string
}

func (o *output) Error() string {
	ret := o.description
	if o.usage != "" {
		ret += fmt.Sprintf("\nUsage: %s", o.usage)
	}
	if o.suggestion != "" {
		ret += fmt.Sprintf("\n%s", o.suggestion)
	}
	return ret
}

// NewOuput creates a new error for CLI output.
func NewOuput(description, usage, suggestion string) error {
	return &output{
		description: description,
		usage:       usage,
		suggestion:  suggestion,
	}
}

// NewErrEmptyTagOrDigest creates a new error based on the reference string.
func NewErrEmptyTagOrDigest(ref registry.Reference) error {
	return NewErrEmptyTagOrDigestStr(ref.String())
}

// NewErrEmptyTagOrDigestStr creates a new error based on the reference string.
func NewErrEmptyTagOrDigestStr(ref string) error {
	return fmt.Errorf("%q: no tag or digest when expecting <name:tag|name@digest>", ref)
}
