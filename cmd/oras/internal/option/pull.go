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

// Pull option struct.
type Pull struct {
	KeepOldFiles      bool
	PathTraversal     bool
	Output            string
	ManifestConfigRef string
}

// ApplyFlags applies flags to a command flag set.
func (pull *Pull) ApplyFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&pull.KeepOldFiles, "keep-old-files", "k", false, "do not replace existing files when pulling, treat them as errors")
	fs.BoolVarP(&pull.PathTraversal, "allow-path-traversal", "T", false, "allow storing files out of the output directory")
	fs.StringVarP(&pull.Output, "output", "o", "", "output directory")
	fs.StringVarP(&pull.ManifestConfigRef, "manifest-config", "", "", "output manifest config file")
}
