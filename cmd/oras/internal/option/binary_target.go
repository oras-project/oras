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
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
)

// BinaryTarget struct contains flags and arguments specifying two registries or
// image layouts.
// BinaryTarget implements errors.Handler interface.
type BinaryTarget struct {
	From        Target
	To          Target
	resolveFlag []string
}

// EnsureSourceTargetReferenceNotEmpty ensures that from target reference is not empty.
func (target *BinaryTarget) EnsureSourceTargetReferenceNotEmpty(cmd *cobra.Command) error {
	if target.From.Reference == "" {
		return oerrors.NewErrEmptyTagOrDigest(target.From.RawReference, cmd, true)
	}
	return nil
}

// EnableDistributionSpecFlag set distribution specification flag as applicable.
func (target *BinaryTarget) EnableDistributionSpecFlag() {
	target.From.EnableDistributionSpecFlag()
	target.To.EnableDistributionSpecFlag()
}

// ApplyFlags applies flags to a command flag set fs.
func (target *BinaryTarget) ApplyFlags(fs *pflag.FlagSet) {
	target.From.ApplyFlagsWithPrefix(fs, "from", "source")
	target.To.ApplyFlagsWithPrefix(fs, "to", "destination")
	fs.StringArrayVarP(&target.resolveFlag, "resolve", "", nil, "base DNS rules formatted in `host:port:address[:address_port]` for --from-resolve and --to-resolve")
}

// Parse parses user-provided flags and arguments into option struct.
func (target *BinaryTarget) Parse(cmd *cobra.Command) error {
	target.From.warned = make(map[string]*sync.Map)
	target.To.warned = target.From.warned
	// resolve are parsed in array order, latter will overwrite former
	target.From.resolveFlag = append(target.resolveFlag, target.From.resolveFlag...)
	target.To.resolveFlag = append(target.resolveFlag, target.To.resolveFlag...)
	return Parse(cmd, target)
}

// Modify handles error during cmd execution.
func (target *BinaryTarget) Modify(cmd *cobra.Command, err error) (error, bool) {
	if modifiedErr, modified := target.From.Modify(cmd, err); modified {
		return modifiedErr, modified
	}
	return target.To.Modify(cmd, err)
}
