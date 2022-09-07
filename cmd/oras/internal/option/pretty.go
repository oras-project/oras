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
	"io"

	"github.com/spf13/pflag"
)

// Pretty option struct.
type Pretty struct {
	pretty bool
}

// ApplyFlags applies flags to a command flag set.
func (opts *Pretty) ApplyFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&opts.pretty, "pretty", "", false, "prettify JSON objects printed to stdout")
}

// Output outputs the prettified content if `--pretty` flag is used. Otherwise
// outputs the original content.
func (opts *Pretty) Output(w io.Writer, content []byte) error {
	if opts.pretty {
		buf := bytes.NewBuffer(nil)
		if err := json.Indent(buf, content, "", "  "); err != nil {
			return fmt.Errorf("failed to prettify: %w", err)
		}
		buf.WriteByte('\n')
		content = buf.Bytes()
	}

	_, err := w.Write(content)
	return err
}
