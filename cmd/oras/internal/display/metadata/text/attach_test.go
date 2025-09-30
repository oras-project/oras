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
	"os"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
)

func TestNewAttachHandler(t *testing.T) {
	printer := output.NewPrinter(&bytes.Buffer{}, os.Stderr)
	handler := NewAttachHandler(printer)

	if handler == nil {
		t.Fatal("NewAttachHandler() returned nil")
	}

	attachHandler, ok := handler.(*AttachHandler)
	if !ok {
		t.Fatal("NewAttachHandler() did not return an *AttachHandler")
	}

	if attachHandler.printer != printer {
		t.Error("NewAttachHandler() did not set printer correctly")
	}
}

func TestAttachHandler_OnAttached(t *testing.T) {
	content := []byte("test content")
	subjectDigest := digest.FromBytes(content)
	rootDigest := digest.FromBytes([]byte("root content"))

	tests := []struct {
		name                     string
		target                   *option.Target
		root                     ocispec.Descriptor
		subject                  ocispec.Descriptor
		expectedDisplayReference string
	}{
		{
			name: "reference ends with subject digest",
			target: &option.Target{
				Type:         "registry",
				RawReference: "example.com/repo@" + subjectDigest.String(),
				Path:         "example.com/repo",
			},
			root: ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.manifest.v1+json",
				Digest:    rootDigest,
				Size:      100,
			},
			subject: ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.manifest.v1+json",
				Digest:    subjectDigest,
				Size:      50,
			},
			expectedDisplayReference: "[registry] example.com/repo@" + subjectDigest.String(),
		},
		{
			name: "reference with tag, not digest",
			target: &option.Target{
				Type:         "registry",
				RawReference: "example.com/repo:latest",
				Path:         "example.com/repo",
			},
			root: ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.manifest.v1+json",
				Digest:    rootDigest,
				Size:      100,
			},
			subject: ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.manifest.v1+json",
				Digest:    subjectDigest,
				Size:      50,
			},
			expectedDisplayReference: "[registry] example.com/repo@" + subjectDigest.String(),
		},
		{
			name: "reference with partial digest match",
			target: &option.Target{
				Type:         "registry",
				RawReference: "example.com/repo@sha256:partial" + subjectDigest.String()[12:],
				Path:         "example.com/repo",
			},
			root: ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.manifest.v1+json",
				Digest:    rootDigest,
				Size:      100,
			},
			subject: ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.manifest.v1+json",
				Digest:    subjectDigest,
				Size:      50,
			},
			expectedDisplayReference: "[registry] example.com/repo@" + subjectDigest.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := output.NewPrinter(&bytes.Buffer{}, os.Stderr)
			handler := &AttachHandler{
				printer: printer,
			}

			handler.OnAttached(tt.target, tt.root, tt.subject)

			if handler.root.Digest != tt.root.Digest {
				t.Errorf("OnAttached() root digest = %v, want %v", handler.root.Digest, tt.root.Digest)
			}

			if handler.subjectDisplayReference != tt.expectedDisplayReference {
				t.Errorf("OnAttached() subjectDisplayReference = %v, want %v", handler.subjectDisplayReference, tt.expectedDisplayReference)
			}
		})
	}
}

func TestAttachHandler_Render(t *testing.T) {
	content := []byte("test content")
	rootDigest := digest.FromBytes(content)

	tests := []struct {
		name                    string
		out                     *bytes.Buffer
		errorOut                bool
		subjectDisplayReference string
		root                    ocispec.Descriptor
		wantErr                 bool
		expectedOutput          string
	}{
		{
			name:                    "successful render",
			out:                     &bytes.Buffer{},
			subjectDisplayReference: "[registry] example.com/repo:latest",
			root: ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.manifest.v1+json",
				Digest:    rootDigest,
				Size:      int64(len(content)),
			},
			wantErr:        false,
			expectedOutput: "Attached to [registry] example.com/repo:latest\nDigest: " + rootDigest.String() + "\n",
		},
		{
			name:                    "error on first print",
			out:                     nil,
			errorOut:                true,
			subjectDisplayReference: "[registry] example.com/repo:latest",
			root: ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.manifest.v1+json",
				Digest:    rootDigest,
				Size:      int64(len(content)),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var printer *output.Printer
			if tt.errorOut {
				printer = output.NewPrinter(&errorWriter{}, os.Stderr)
			} else {
				printer = output.NewPrinter(tt.out, os.Stderr)
			}

			handler := &AttachHandler{
				printer:                 printer,
				subjectDisplayReference: tt.subjectDisplayReference,
				root:                    tt.root,
			}

			err := handler.Render()

			if (err != nil) != tt.wantErr {
				t.Errorf("AttachHandler.Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.out != nil {
				output := tt.out.String()
				if output != tt.expectedOutput {
					t.Errorf("AttachHandler.Render() output = %q, want %q", output, tt.expectedOutput)
				}
			}
		})
	}
}

func TestAttachHandler_Render_SuccessfulCase(t *testing.T) {
	content := []byte("test content")
	rootDigest := digest.FromBytes(content)

	buffer := &bytes.Buffer{}
	printer := output.NewPrinter(buffer, os.Stderr)

	handler := &AttachHandler{
		printer:                 printer,
		subjectDisplayReference: "[registry] example.com/repo:latest",
		root: ocispec.Descriptor{
			MediaType: "application/vnd.oci.image.manifest.v1+json",
			Digest:    rootDigest,
			Size:      int64(len(content)),
		},
	}

	err := handler.Render()
	if err != nil {
		t.Errorf("AttachHandler.Render() error = %v, want nil", err)
	}

	expectedOutput := "Attached to [registry] example.com/repo:latest\nDigest: " + rootDigest.String() + "\n"
	if buffer.String() != expectedOutput {
		t.Errorf("AttachHandler.Render() output = %q, want %q", buffer.String(), expectedOutput)
	}
}

func TestAttachHandler_InterfaceCompliance(t *testing.T) {
	var _ metadata.AttachHandler = (*AttachHandler)(nil)
}
