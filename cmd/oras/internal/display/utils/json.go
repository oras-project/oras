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

package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// PrintPrettyJSON prints the object to the writer in JSON format.
func PrintPrettyJSON(out io.Writer, object any) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(object)
}

// PrintJSON writes the data to the output stream, optionally prettifying it.
func PrintJSON(out io.Writer, data []byte, pretty bool) error {
	if pretty {
		buf := bytes.NewBuffer(nil)
		if err := json.Indent(buf, data, "", "  "); err != nil {
			return fmt.Errorf("failed to prettify: %w", err)
		}
		buf.WriteByte('\n')
		data = buf.Bytes()
	}
	_, err := out.Write(data)
	return err
}
