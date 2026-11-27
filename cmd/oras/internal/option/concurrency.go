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
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const defaultConcurrency = 3

// Concurrency option struct.
type Concurrency struct {
	Concurrency int
}

// ApplyFlags applies flags to a command flag set.
func (opts *Concurrency) ApplyFlags(fs *pflag.FlagSet) {
	if opts.Concurrency <= 0 {
		opts.Concurrency = defaultConcurrency
	}
	fs.IntVarP(&opts.Concurrency, "concurrency", "", opts.Concurrency, "concurrency level")
}

// Parse parses the input concurrency flag.
func (opts *Concurrency) Parse(*cobra.Command) error {
	if opts.Concurrency < 1 {
		return errors.New("concurrency value should be at least 1")
	}
	return nil
}
