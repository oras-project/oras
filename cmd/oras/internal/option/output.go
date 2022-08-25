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
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

// Output option struct.
type Output struct {
	OutputDescriptor bool
	Pretty           bool
}

// ApplyFlags applies flags to a command flag set.
func (opts *Output) ApplyFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&opts.OutputDescriptor, "descriptor", "", false, "output descriptor")
	fs.BoolVarP(&opts.Pretty, "pretty", "", false, "output prettified content")
}

// OutputContent outputs the content to stdout
func (opts *Output) OutputContent(content []byte) error {
	if opts.Pretty {
		buf := bytes.NewBuffer(nil)
		if err := json.Indent(buf, content, "", "  "); err != nil {
			return fmt.Errorf("failed to prettify: %w", err)
		}
		buf.WriteByte('\n')
		content = buf.Bytes()
	}

	_, err := os.Stdout.Write(content)
	return err
}
