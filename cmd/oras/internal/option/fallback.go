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

import "github.com/spf13/pflag"

type FallbackRemote struct {
	Remote
	distributionSpec
}

// ApplyFlags applies flags to a command flag set.
func (opts *FallbackRemote) ApplyFlags(fs *pflag.FlagSet) {
	opts.Remote.ApplyFlags(fs)
	opts.distributionSpec.ApplyFlags(fs)
}

// Parse tries to read password with optional cmd prompt.
func (opts *FallbackRemote) Parse() error {
	return Parse(opts)
}

// ApplyFlagsWithPrefix applies flags to a command flag set with a prefix string.
func (opts *FallbackRemote) ApplyFlagsWithPrefix(fs *pflag.FlagSet, prefix, description string) {
	opts.Remote.ApplyFlagsWithPrefix(fs, prefix, description)
}

type FallbackTarget struct {
	Target
	distributionSpec
}

// ApplyFlags applies flags to a command flag set.
func (opts *FallbackTarget) ApplyFlags(fs *pflag.FlagSet) {
	opts.Target.ApplyFlags(fs)
	opts.distributionSpec.ApplyFlags(fs)
}

// Parse tries to read password with optional cmd prompt.
func (opts *FallbackTarget) Parse() error {
	return Parse(opts)
}

// ApplyFlagsWithPrefix applies flags to a command flag set with a prefix string.
func (opts *FallbackTarget) ApplyFlagsWithPrefix(fs *pflag.FlagSet, prefix, description string) {
	opts.Target.ApplyFlagsWithPrefix(fs, prefix, description)
}
