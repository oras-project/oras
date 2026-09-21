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
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

var fetchTestManifest = []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json"}`)

// failingWriter fails every write, to exercise the output error path.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write error")
}

// --pretty is meaningless for a file, so the handler drops it rather than
// writing indented JSON to disk.
func TestNewManifestFetchHandler_prettyIgnoredForFile(t *testing.T) {
	tests := []struct {
		name       string
		outputPath string
		wantPretty bool
	}{
		{"file path drops pretty", filepath.Join(t.TempDir(), "manifest.json"), false},
		{"stdout keeps pretty", "-", true},
		{"empty path keeps pretty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, ok := NewManifestFetchHandler(&bytes.Buffer{}, true, tt.outputPath).(*manifestFetch)
			if !ok {
				t.Fatal("NewManifestFetchHandler() did not return a *manifestFetch")
			}
			if handler.pretty != tt.wantPretty {
				t.Errorf("NewManifestFetchHandler() pretty = %v, want %v", handler.pretty, tt.wantPretty)
			}
			if handler.outputPath != tt.outputPath {
				t.Errorf("NewManifestFetchHandler() outputPath = %q, want %q", handler.outputPath, tt.outputPath)
			}
		})
	}
}

func TestManifestFetch_OnContentFetched_stdout(t *testing.T) {
	tests := []struct {
		name       string
		pretty     bool
		outputPath string
	}{
		{"compact", false, ""},
		{"compact to dash", false, "-"},
		{"pretty", true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			handler := NewManifestFetchHandler(buf, tt.pretty, tt.outputPath)

			if err := handler.OnContentFetched(ocispec.Descriptor{}, fetchTestManifest); err != nil {
				t.Fatalf("OnContentFetched() error = %v, want nil", err)
			}

			out := buf.String()
			if !strings.Contains(out, `"schemaVersion"`) {
				t.Errorf("OnContentFetched() = %q, want it to contain the manifest", out)
			}
			// Only the pretty form is indented and newline-terminated; the
			// compact form stays byte-for-byte what was fetched so it can be
			// piped.
			if indented := strings.Contains(out, "\n  "); indented != tt.pretty {
				t.Errorf("OnContentFetched() indented = %v, want %v; output %q", indented, tt.pretty, out)
			}
			if !tt.pretty && out != string(fetchTestManifest) {
				t.Errorf("OnContentFetched() = %q, want the manifest unchanged", out)
			}
		})
	}
}

func TestManifestFetch_OnContentFetched_file(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	stdout := &bytes.Buffer{}
	// pretty is requested but must be ignored for a file.
	handler := NewManifestFetchHandler(stdout, true, path)

	if err := handler.OnContentFetched(ocispec.Descriptor{}, fetchTestManifest); err != nil {
		t.Fatalf("OnContentFetched() error = %v, want nil", err)
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

func TestManifestFetch_OnContentFetched_createError(t *testing.T) {
	// A path under a file rather than a directory cannot be created.
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0600); err != nil {
		t.Fatalf("cannot create the blocking file: %v", err)
	}
	handler := NewManifestFetchHandler(&bytes.Buffer{}, false, filepath.Join(blocker, "manifest.json"))

	err := handler.OnContentFetched(ocispec.Descriptor{}, fetchTestManifest)
	if err == nil {
		t.Fatal("OnContentFetched() error = nil, want an error when the file cannot be created")
	}
	if !strings.Contains(err.Error(), "failed to open") {
		t.Errorf("OnContentFetched() error = %q, want it to mention that opening failed", err)
	}
}

func TestManifestFetch_OnContentFetched_writeError(t *testing.T) {
	handler := NewManifestFetchHandler(failingWriter{}, false, "")

	if err := handler.OnContentFetched(ocispec.Descriptor{}, fetchTestManifest); err == nil {
		t.Error("OnContentFetched() error = nil, want an error when the writer fails")
	}
}

// Prettifying runs on the fetched bytes, so content that is not JSON is
// reported rather than written out half-formed.
func TestManifestFetch_OnContentFetched_invalidJSON(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewManifestFetchHandler(buf, true, "")

	err := handler.OnContentFetched(ocispec.Descriptor{}, []byte("not json"))
	if err == nil {
		t.Fatal("OnContentFetched() error = nil, want an error for content that is not JSON")
	}
	if !strings.Contains(err.Error(), "prettify") {
		t.Errorf("OnContentFetched() error = %q, want it to mention prettifying", err)
	}
}
