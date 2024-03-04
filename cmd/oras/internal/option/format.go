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

// Format is a flag to format metadata into output.
type Format struct {
	Template string
}

// ApplyFlag implements FlagProvider.ApplyFlag.
func (opts *Format) ApplyFlags(fs *pflag.FlagSet) {
	const name = "format"
	if fs.Lookup(name) != nil {
		// allow command to overwrite the flag
		return
	}
	fs.StringVar(&opts.Template, name, "", `[Experimental] Format output using a custom template:
'json':       Print in JSON format
'$TEMPLATE':  Print output using the given Go template.`)
}
