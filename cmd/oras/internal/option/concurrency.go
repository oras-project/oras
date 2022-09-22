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
	"github.com/spf13/pflag"
	"oras.land/oras-go/v2"
)

// Concurrency option struct.
type Concurrency struct {
	Concurrency int64
}

// ApplyFlags applies flags to a command flag set.
func (opts *Concurrency) ApplyFlags(fs *pflag.FlagSet) {
	fs.Int64VarP(&opts.Concurrency, "concurrency", "", 3, "provide concurrency number")
}

// TagNOptions changes the Concurrency number for oras.TagN method.
func (opts *Concurrency) TagNOptions() oras.TagNOptions {
	tagNOpt := oras.DefaultTagNOptions
	tagNOpt.Concurrency = opts.Concurrency
	return tagNOpt
}

// TagBytesNOptions changes the Concurrency number for oras.TagBytesN method.
func (opts *Concurrency) TagBytesNOptions() oras.TagBytesNOptions {
	tagNOpt := oras.DefaultTagBytesNOptions
	tagNOpt.Concurrency = opts.Concurrency
	return tagNOpt
}
