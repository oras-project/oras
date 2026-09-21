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

package content

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_manifestIndexCreate_OnContentCreated(t *testing.T) {
	testHandler := NewManifestIndexCreateHandler(os.Stdout, false, "invalid/path")
	if err := testHandler.OnContentCreated([]byte("test content")); err == nil {
		t.Errorf("manifestIndexCreate.OnContentCreated() error = %v, wantErr non-nil error", err)
	}
}

// --pretty is meaningless for a file, so the handler drops it rather than
// writing indented JSON to disk.
func TestNewManifestIndexCreateHandler_prettyIgnoredForFile(t *testing.T) {
	tests := []struct {
		name       string
		outputPath string
		wantPretty bool
	}{
		{"file path drops pretty", filepath.Join(t.TempDir(), "index.json"), false},
		{"stdout keeps pretty", "-", true},
		{"empty path keeps pretty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, ok := NewManifestIndexCreateHandler(&bytes.Buffer{}, true, tt.outputPath).(*manifestIndexCreate)
			if !ok {
				t.Fatal("NewManifestIndexCreateHandler() did not return a *manifestIndexCreate")
			}
			if handler.pretty != tt.wantPretty {
				t.Errorf("NewManifestIndexCreateHandler() pretty = %v, want %v", handler.pretty, tt.wantPretty)
			}
			if handler.outputPath != tt.outputPath {
				t.Errorf("NewManifestIndexCreateHandler() outputPath = %q, want %q", handler.outputPath, tt.outputPath)
			}
		})
	}
}

func Test_manifestIndexCreate_OnContentCreated_stdout(t *testing.T) {
	tests := []struct {
		name   string
		pretty bool
	}{
		{"compact", false},
		{"pretty", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			handler := NewManifestIndexCreateHandler(buf, tt.pretty, "")

			if err := handler.OnContentCreated(fetchTestManifest); err != nil {
				t.Fatalf("OnContentCreated() error = %v, want nil", err)
			}

			out := buf.String()
			if indented := strings.Contains(out, "\n  "); indented != tt.pretty {
				t.Errorf("OnContentCreated() indented = %v, want %v; output %q", indented, tt.pretty, out)
			}
			if !tt.pretty && out != string(fetchTestManifest) {
				t.Errorf("OnContentCreated() = %q, want the content unchanged", out)
			}
		})
	}
}

func Test_manifestIndexCreate_OnContentCreated_file(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.json")
	stdout := &bytes.Buffer{}
	// pretty is requested but must be ignored for a file.
	handler := NewManifestIndexCreateHandler(stdout, true, path)

	if err := handler.OnContentCreated(fetchTestManifest); err != nil {
		t.Fatalf("OnContentCreated() error = %v, want nil", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read the written file: %v", err)
	}
	if !bytes.Equal(got, fetchTestManifest) {
		t.Errorf("file content = %q, want %q", got, fetchTestManifest)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want nothing written when an output path is given", stdout.String())
	}
}

func Test_manifestIndexCreate_OnContentCreated_writeError(t *testing.T) {
	handler := NewManifestIndexCreateHandler(failingWriter{}, false, "")

	if err := handler.OnContentCreated(fetchTestManifest); err == nil {
		t.Error("OnContentCreated() error = nil, want an error when the writer fails")
	}
}
