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

package option

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"
	"oras.land/oras/cmd/oras/internal/output"
)

const NoTTYFlag = "no-tty"

// Common option struct.
type Common struct {
	Debug   bool
	Verbose bool // deprecated, the current default behavior is equivalent to verbose=true TODO: better doc
	TTY     *os.File
	*output.Printer
	noTTY            bool
	SuppressUntitled bool // equivalent to verbose=false TODO: better doc
}

// ApplyFlags applies flags to a command flag set.
func (opts *Common) ApplyFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&opts.Debug, "debug", "d", false, "output debug logs (implies --no-tty)")
	fs.BoolVarP(&opts.Verbose, "verbose", "v", false, "[Deprecated] verbose output")
	fs.BoolVarP(&opts.noTTY, NoTTYFlag, "", false, "[Preview] do not show progress output")

	fs.MarkDeprecated("verbose", "and may be removed in a future release.") // TODO: remove -v in e2e test; test deprecation message
}

// Parse gets target options from user input.
func (opts *Common) Parse(cmd *cobra.Command) error {
	opts.Printer = output.NewPrinter(cmd.OutOrStdout(), cmd.OutOrStderr(), opts.SuppressUntitled)
	// use STDERR as TTY output since STDOUT is reserved for pipeable output
	return opts.parseTTY(os.Stderr)
}

// parseTTY gets target options from user input.
func (opts *Common) parseTTY(f *os.File) error {
	if !opts.noTTY {
		if opts.Debug {
			opts.noTTY = true
		} else if term.IsTerminal(int(f.Fd())) {
			opts.TTY = f
		}
	}
	return nil
}

// UpdateTTY updates the TTY value, given the status of --no-tty flag and output
// path value.
func (opts *Common) UpdateTTY(flagPresent bool, toSTDOUT bool) {
	ttyEnforced := flagPresent && !opts.noTTY
	if opts.noTTY || (toSTDOUT && !ttyEnforced) {
		opts.TTY = nil
	}
}
