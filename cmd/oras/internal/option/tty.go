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

// TTY option struct.
type TTY struct {
	noTTY bool
	Tty   *os.File
}

// ApplyFlags applies flags to a command flag set.
func (opts *TTY) ApplyFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&opts.noTTY, NoTTYFlag, "", false, "[Preview] disable progress bars")
}

// Parse parses the input notty flag.
func (opts *TTY) Parse(*cobra.Command) error {
	// use STDERR as TTY output since STDOUT is reserved for pipeable output
	if !opts.noTTY {
		f := os.Stderr
		if term.IsTerminal(int(f.Fd())) {
			opts.Tty = f
		}
	}
	return nil
}

// UpdateTTY updates the TTY value, given the status of --debug flag, --no-tty flag and output
// path value.TTY value is set to nil if
// 1. --no-tty flag is set to true
// 2. --debug flag is used
// 3. output path is set to stdout and --no-tty flag is not explicitly set to false
// (i.e. not --no-tty==false)
func (opts *TTY) UpdateTTY(debugEnabled, nottyFlagPresent, toSTDOUT bool) {
	ttyEnforced := nottyFlagPresent && !opts.noTTY
	if debugEnabled || opts.noTTY || (toSTDOUT && !ttyEnforced) {
		opts.Tty = nil
	}
}
