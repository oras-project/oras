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
)

const NoTTYFlag = "no-tty"

// ???.
type NoTTY struct {
	// ???
	TTY *os.File

	// ???
	noTTY bool
}

// ApplyFlags applies flags to a command flag set.
func (opts *NoTTY) ApplyFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&opts.noTTY, NoTTYFlag, "", false, "[Preview] disable progress bars")
}

// Parse parses the input annotation flags.
func (opts *NoTTY) Parse(*cobra.Command) error {
	// ??? need to handle this in each command
	if !opts.noTTY {
		// if opts.Debug {
		// 	opts.noTTY = true // ??? need to handle this in each command
		// } else if term.IsTerminal(int(f.Fd())) {
		// 	opts.TTY = f
		// }
		f := os.Stderr // use STDERR as TTY output since STDOUT is reserved for pipeable output
		if term.IsTerminal(int(f.Fd())) {
			opts.TTY = f
		}
	}
	return nil
}

// UpdateTTY updates the TTY value, given the status of --no-tty flag and output
// path value.
func (opts *NoTTY) UpdateTTY(flagPresent bool, toSTDOUT bool) {
	ttyEnforced := flagPresent && !opts.noTTY
	if opts.noTTY || (toSTDOUT && !ttyEnforced) {
		opts.TTY = nil
	}
}
