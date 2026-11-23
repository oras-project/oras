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

package output

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

// ToMap converts the data to a map[string]any with json tag as key.
// Integral numbers are converted to int64 instead of float64, so that large
// values such as blob sizes are not printed in scientific notation or rounded
// when they are rendered by a template.
func ToMap(data any) (map[string]any, error) {
	// slow but easy
	content, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()
	var ret map[string]any
	if err = decoder.Decode(&ret); err != nil {
		return nil, err
	}
	for k, v := range ret {
		ret[k] = convertNumbers(v)
	}
	return ret, nil
}

// convertNumbers replaces every json.Number in v with an int64 when the number
// is an integer that fits in an int64, or with a float64 otherwise.
func convertNumbers(v any) any {
	switch v := v.(type) {
	case map[string]any:
		for key, value := range v {
			v[key] = convertNumbers(value)
		}
		return v
	case []any:
		for i, value := range v {
			v[i] = convertNumbers(value)
		}
		return v
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i
		}
		if f, err := v.Float64(); err == nil {
			return f
		}
		return v
	}
	return v
}
