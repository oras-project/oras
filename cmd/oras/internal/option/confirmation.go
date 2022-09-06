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
	"fmt"
	"strings"

	"github.com/spf13/pflag"
)

// Confirmation option struct.
type Confirmation struct {
	Confirmed bool
}

var scanln func(a ...any) (n int, err error) = fmt.Scanln

// ApplyFlags applies flags to a command flag set.
func (opts *Confirmation) ApplyFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&opts.Confirmed, "yes", "y", false, "do not prompt for confirmation")
}

func (opts *Confirmation) AskForConfirmation(message string) (bool, error) {
	if opts.Confirmed {
		return true, nil
	}

	for {
		fmt.Print(message)

		var response string
		_, err := scanln(&response)
		if err != nil {
			return false, err
		}

		switch strings.ToLower(response) {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		}
	}
}
