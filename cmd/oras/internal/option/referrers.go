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

	"github.com/spf13/pflag"
	"oras.land/oras-go/v2/registry/remote"
)

// Referrers option struct.
type Referrers struct {
	GC bool
}

// ApplyFlags applies flags to a command flag set.
func (opts *Referrers) ApplyFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&opts.GC, "referrers-gc", "", false, "enforce garbage collection when referrers API is not supported")
}

// SetReferrersGC sets the referrers GC option for the passed-in target.
func (opts *Referrers) SetReferrersGC(target any) error {
	if repo, ok := target.(*remote.Repository); ok {
		repo.SkipReferrersGC = !opts.GC
	} else if opts.GC {
		return errors.New("referrers GC can only be enforced to registry targets")
	}
	return nil
}
