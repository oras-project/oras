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
	"encoding/json"
	"io"

	"bytes"
	"encoding/json"
	"fmt"
	"io"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// GenerateContentKey generates a unique key for each content descriptor, using
// its digest and name if applicable.
func GenerateContentKey(desc v1.Descriptor) string {
	return desc.Digest.String() + desc.Annotations[v1.AnnotationTitle]
}
)

// Prompt constants for pull.
const (
	PullPromptDownloading = "Downloading"
	PullPromptPulled      = "Pulled     "
	PullPromptProcessing  = "Processing "
	PullPromptSkipped     = "Skipped    "
	PullPromptRestored    = "Restored   "
	PullPromptDownloaded  = "Downloaded "
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
