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

package descriptor

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const testDigest = "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

var testDescriptor = ocispec.Descriptor{
	MediaType: "application/vnd.oci.image.manifest.v1+json",
	Digest:    testDigest,
	Size:      100,
}

// errWriter fails every write, to exercise the output error path.
type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, errors.New("write error")
}

func TestNewManifestFetchHandler(t *testing.T) {
	if got := NewManifestFetchHandler(&bytes.Buffer{}, false); got == nil {
		t.Fatal("NewManifestFetchHandler() returned nil")
	}
}

func TestManifestFetchHandler_OnFetched(t *testing.T) {
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
			handler := NewManifestFetchHandler(buf, tt.pretty)

			if err := handler.OnFetched("localhost:5000/test:v1", testDescriptor, nil); err != nil {
				t.Fatalf("OnFetched() error = %v, want nil", err)
			}

			// The output must round-trip back to the same descriptor whether or
			// not it is indented.
			var got ocispec.Descriptor
			if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
				t.Fatalf("OnFetched() wrote invalid JSON %q: %v", buf.String(), err)
			}
			if !reflect.DeepEqual(got, testDescriptor) {
				t.Errorf("OnFetched() descriptor = %v, want %v", got, testDescriptor)
			}

			// Only the pretty form is indented and newline-terminated; the
			// compact form stays on a single line so it can be piped.
			out := buf.String()
			if indented := bytes.Contains(buf.Bytes(), []byte("\n  ")); indented != tt.pretty {
				t.Errorf("OnFetched() indented = %v, want %v; output %q", indented, tt.pretty, out)
			}
			if gotNewline := len(out) > 0 && out[len(out)-1] == '\n'; gotNewline != tt.pretty {
				t.Errorf("OnFetched() trailing newline = %v, want %v; output %q", gotNewline, tt.pretty, out)
			}
		})
	}
}

func TestManifestFetchHandler_OnFetched_writeError(t *testing.T) {
	handler := NewManifestFetchHandler(errWriter{}, false)
	if err := handler.OnFetched("localhost:5000/test:v1", testDescriptor, nil); err == nil {
		t.Error("OnFetched() error = nil, want an error when the writer fails")
	}
}

func TestManifestFetchHandler_OnFetched_ignoresNameAndContent(t *testing.T) {
	// OnFetched renders only the descriptor: the reference and the raw manifest
	// bytes are accepted but must not reach the output.
	buf := &bytes.Buffer{}
	handler := NewManifestFetchHandler(buf, false)

	if err := handler.OnFetched("localhost:5000/ignored:tag", testDescriptor, []byte(`{"ignored":true}`)); err != nil {
		t.Fatalf("OnFetched() error = %v, want nil", err)
	}
	for _, unwanted := range []string{"ignored", "localhost:5000"} {
		if bytes.Contains(buf.Bytes(), []byte(unwanted)) {
			t.Errorf("OnFetched() output %q should not contain %q", buf.String(), unwanted)
		}
	}
}
