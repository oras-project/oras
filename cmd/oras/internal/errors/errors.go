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

	"github.com/spf13/cobra"
	"oras.land/oras-go/v2/registry"
)

// Error is the error type for CLI error messaging.
type Error struct {
	Err            error
	Usage          string
	Recommendation string
}

// Unwrap implements the errors.Wrapper interface.
func (o *Error) Unwrap() error {
	return o.Err
}

// Error implements the error interface.
func (o *Error) Error() string {
	ret := o.Err.Error()
	if o.Usage != "" {
		ret += fmt.Sprintf("\nUsage: %s", o.Usage)
	}
	if o.Recommendation != "" {
		ret += fmt.Sprintf("\n%s", o.Recommendation)
	}
	return ret
}

// CheckArgs checks the args with the checker function.
func CheckArgs(checker func(args []string) (bool, string), Usage string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if ok, text := checker(args); !ok {
			return &Error{
				Err:            fmt.Errorf(`%q requires %s but got %d`, cmd.CommandPath(), text, len(args)),
				Usage:          fmt.Sprintf("%s %s", cmd.Parent().CommandPath(), cmd.Use),
				Recommendation: fmt.Sprintf(`Please specify %s as %s. Run "%s -h" for more options and examples`, text, Usage, cmd.CommandPath()),
			}
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
