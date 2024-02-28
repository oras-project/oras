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
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/errcode"
)

// RegistryErrorPrefix is the commandline prefix for errors from registry.
const RegistryErrorPrefix = "Error response from registry:"

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

// Modifier modifies the error during cmd execution.
type Modifier interface {
	Modify(cmd *cobra.Command, err error) (modifiedErr error, modified bool)
}

// Command returns an error-handled cobra command.
func Command(cmd *cobra.Command, handler Modifier) *cobra.Command {
	runE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		err := runE(cmd, args)
		if err != nil {
			err, _ = handler.Modify(cmd, err)
			return err
		}
		return nil
	}
	return cmd
}

// TrimErrResp tries to trim toTrim from err.
func TrimErrResp(err error, toTrim error) error {
	var inner error
	if errResp, ok := toTrim.(*errcode.ErrorResponse); ok {
		if len(errResp.Errors) == 0 {
			return fmt.Errorf("recognizable error message not found: %w", toTrim)
		}
		inner = errResp.Errors
	} else {
		return err
	}
	return reWrap(err, toTrim, inner)
}

// TrimErrBasicCredentialNotFound trims the credentials from err.
// Caller should make sure the err is auth.ErrBasicCredentialNotFound.
func TrimErrBasicCredentialNotFound(err error) error {
	toTrim := err
	inner := err
	for {
		switch x := inner.(type) {
		case interface{ Unwrap() error }:
			toTrim = inner
			inner = x.Unwrap()
			continue
		case interface{ Unwrap() []error }:
			for _, errItem := range x.Unwrap() {
				if errors.Is(errItem, auth.ErrBasicCredentialNotFound) {
					toTrim = errItem
					inner = errItem
					break
				}
			}
			continue
		}
		break
	}
	return reWrap(err, toTrim, auth.ErrBasicCredentialNotFound)
}

// reWrap re-wraps errA to errC and trims out errB, returns errC if scrub fails.
// +---------- errA ----------+
// |         +---- errB ----+ |      +---- errA ----+
// |         |    errC      | |  =>  |     errC     |
// |         +--------------+ |      +--------------+
// +--------------------------+
func reWrap(errA, errB, errC error) error {
	// TODO: trim dedicated error type when
	// https://github.com/oras-project/oras-go/issues/677 is done
	contentA := errA.Error()
	contentB := errB.Error()
	if idx := strings.Index(contentA, contentB); idx > 0 {
		return fmt.Errorf("%s%w", contentA[:idx], errC)
	}
	return errC
}

// NewErrEmptyTagOrDigest creates a new error based on the reference string.
func NewErrEmptyTagOrDigest(ref string, cmd *cobra.Command, needsTag bool) error {
	form := `"<name>@<digest>"`
	errMsg := `no digest specified`
	if needsTag {
		form = fmt.Sprintf(`"<name>:<tag>" or %s`, form)
		errMsg = "no tag or digest specified"
	}
	return &Error{
		Err:            fmt.Errorf(`"%s": %s`, ref, errMsg),
		Usage:          fmt.Sprintf("%s %s", cmd.Parent().CommandPath(), cmd.Use),
		Recommendation: fmt.Sprintf(`Please specify a reference in the form of %s. Run "%s -h" for more options and examples`, form, cmd.CommandPath()),
	}
}
