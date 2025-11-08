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

package text

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
)

func TestNewTagHandler(t *testing.T) {
	buf := &bytes.Buffer{}
	printer := output.NewPrinter(buf, os.Stderr)
	target := option.Target{
		Type: "registry",
		Path: "localhost:5000/test",
	}

	handler := NewTagHandler(printer, target)
	
	if handler == nil {
		t.Fatal("NewTagHandler should not return nil")
	}

	th, ok := handler.(*TagHandler)
	if !ok {
		t.Fatal("NewTagHandler should return *TagHandler")
	}

	if th.printer != printer {
		t.Error("printer not set correctly")
	}

	expectedPrefix := fmt.Sprintf("[%s] %s", target.Type, target.Path)
	if th.refPrefix != expectedPrefix {
		t.Errorf("refPrefix = %q, want %q", th.refPrefix, expectedPrefix)
	}
}

func TestTagHandler_OnTagging(t *testing.T) {
	testContent := []byte("hello world")
	testDesc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    digest.FromBytes(testContent),
		Size:      int64(len(testContent)),
	}

	buf := &bytes.Buffer{}
	printer := output.NewPrinter(buf, os.Stderr)
	handler := &TagHandler{
		printer:   printer,
		refPrefix: "[registry] localhost:5000/test",
	}

	err := handler.OnTagging(testDesc, "v1.0")
	if err != nil {
		t.Errorf("OnTagging failed: %v", err)
	}

	expected := "Tagging [registry] localhost:5000/test@" + testDesc.Digest.String() + "\n"
	if buf.String() != expected {
		t.Errorf("output = %q, want %q", buf.String(), expected)
	}

	// Second call shouldn't print anything (printOnce)
	buf.Reset()
	err = handler.OnTagging(testDesc, "v2.0")
	if err != nil {
		t.Errorf("second OnTagging failed: %v", err)
	}
	if buf.Len() > 0 {
		t.Error("OnTagging should only print once")
	}
}

func TestTagHandler_OnTagging_WithError(t *testing.T) {
	testDesc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    digest.FromBytes([]byte("test")),
		Size:      4,
	}

	printer := output.NewPrinter(&errorWriter{}, os.Stderr)
	handler := &TagHandler{
		printer:   printer,
		refPrefix: "[registry] localhost:5000/test",
	}

	err := handler.OnTagging(testDesc, "v1.0")
	if err == nil {
		t.Error("OnTagging should return error when printer fails")
	}
}

func TestTagHandler_OnTagged(t *testing.T) {
	tests := []struct {
		name string
		tag  string
	}{
		{"latest tag", "latest"},
		{"version tag", "v1.2.3"},
		{"custom tag", "production-2024"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			printer := output.NewPrinter(buf, os.Stderr)
			handler := &TagHandler{
				printer:   printer,
				refPrefix: "[registry] localhost:5000/test",
			}

			err := handler.OnTagged(ocispec.Descriptor{}, tc.tag)
			if err != nil {
				t.Errorf("OnTagged failed: %v", err)
			}

			expected := "Tagged " + tc.tag + "\n"
			if buf.String() != expected {
				t.Errorf("output = %q, want %q", buf.String(), expected)
			}
		})
	}
}

func TestTagHandler_OnTagged_WithError(t *testing.T) {
	printer := output.NewPrinter(&errorWriter{}, os.Stderr)
	handler := &TagHandler{
		printer:   printer,
		refPrefix: "[registry] localhost:5000/test",
	}

	err := handler.OnTagged(ocispec.Descriptor{}, "test")
	if err == nil {
		t.Error("OnTagged should return error when printer fails")
	}
}

func TestTagHandler_InterfaceImplementation(t *testing.T) {
	// Verify that TagHandler implements metadata.TagHandler
	var _ metadata.TagHandler = (*TagHandler)(nil)
}
