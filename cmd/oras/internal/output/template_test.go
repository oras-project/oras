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
	"os"
	"testing"
)

func Test_parseAndWrite_err(t *testing.T) {
	if err := ParseAndWrite(os.Stdout, func() {}, ""); err == nil {
		t.Errorf("should return error")
	}
}

func Test_parseAndWrite_integers(t *testing.T) {
	type layer struct {
		Size int64 `json:"size"`
	}
	object := struct {
		Layers []layer `json:"layers"`
	}{
		Layers: []layer{{Size: 5 * 1024 * 1024}, {Size: 10_000_000_000}},
	}
	var buf bytes.Buffer
	if err := ParseAndWrite(&buf, object, "{{range .layers}}{{.size}} {{end}}"); err != nil {
		t.Fatalf("ParseAndWrite() error = %v", err)
	}
	if got, want := buf.String(), "5242880 10000000000 "; got != want {
		t.Errorf("ParseAndWrite() = %q, want %q", got, want)
	}
}
