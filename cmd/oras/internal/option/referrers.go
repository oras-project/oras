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
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"oras.land/oras-go/v2/registry/remote"
)

// Referrers option struct.
type Referrers struct {
	SkipDeleteReferrers bool
}

// ApplyFlags applies flags to a command flag set.
func (opts *Referrers) ApplyFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&opts.SkipDeleteReferrers, "skip-delete-referrers", "", false, "skip deleting old referrers index, only work on registry when referrers API is not supported")
}

// SetReferrersGC sets the referrers GC option for the passed-in target.
func (opts *Referrers) SetReferrersGC(target any, logger logrus.FieldLogger) {
	if repo, ok := target.(*remote.Repository); ok {
		repo.SkipReferrersGC = opts.SkipDeleteReferrers
	} else if opts.SkipDeleteReferrers {
		// not a registry, can't skip referrers deletion
		logger.Warnln("referrers deletion can only be enforced upon registry")
	}
}
