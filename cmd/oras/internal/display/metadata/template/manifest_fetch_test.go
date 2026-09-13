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

package template

import (
	"bytes"
	"testing"
)

func TestNewManifestFetchHandler(t *testing.T) {
	if got := NewManifestFetchHandler(&bytes.Buffer{}, "{{.reference}}"); got == nil {
		t.Fatal("NewManifestFetchHandler() returned nil")
	}
}

func TestManifestFetchHandler_OnFetched(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewManifestFetchHandler(buf, "{{.reference}}|{{.content.schemaVersion}}")

	content := []byte(`{"schemaVersion":2}`)
	if err := handler.OnFetched("localhost:5000/test", testDescriptor, content); err != nil {
		t.Fatalf("OnFetched() error = %v, want nil", err)
	}

	want := "localhost:5000/test@" + testDigest + "|2"
	if got := buf.String(); got != want {
		t.Errorf("OnFetched() = %q, want %q", got, want)
	}
}

func TestManifestFetchHandler_OnFetched_invalidContent(t *testing.T) {
	// Content that is not JSON is not an error: the manifest is reported as
	// null and the descriptor fields are still rendered.
	buf := &bytes.Buffer{}
	handler := NewManifestFetchHandler(buf, "{{.size}}|{{.content}}")

	if err := handler.OnFetched("localhost:5000/test", testDescriptor, []byte("not json")); err != nil {
		t.Fatalf("OnFetched() error = %v, want nil for non-JSON content", err)
	}

	want := "100|<no value>"
	if got := buf.String(); got != want {
		t.Errorf("OnFetched() = %q, want %q", got, want)
	}
}

func TestManifestFetchHandler_OnFetched_invalidTemplate(t *testing.T) {
	handler := NewManifestFetchHandler(&bytes.Buffer{}, "{{.reference")
	if err := handler.OnFetched("localhost:5000/test", testDescriptor, []byte(`{}`)); err == nil {
		t.Error("OnFetched() error = nil, want an error for an unparsable template")
	}
}
