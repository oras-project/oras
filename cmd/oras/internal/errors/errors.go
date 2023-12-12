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
	"strings"

	"github.com/spf13/cobra"
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

// ArgsChecker checks the args with the checker function.
func ArgsChecker(checker func(args []string) (bool, string), usage string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if ok, text := checker(args); !ok {
			return NewOuput(
				fmt.Sprintf(`%q requires %s but got %q`, cmd.CommandPath(), text, strings.Join(args, ",")),
				fmt.Sprintf("%s %s", cmd.Parent().CommandPath(), cmd.Use),
				fmt.Sprintf(`Please specify %s as %s. Run "%s -h" for more options and examples`, text, usage, cmd.CommandPath()),
			)
		}
		return nil
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
