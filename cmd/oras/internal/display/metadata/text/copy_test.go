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
	"io"
	"os"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
)

func TestNewCopyHandler(t *testing.T) {
	tests := []struct {
		name string
		out  io.Writer
	}{
		{
			name: "creates handler with valid printer",
			out:  &bytes.Buffer{},
		},
		{
			name: "creates handler with error writer",
			out:  &errorWriter{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := output.NewPrinter(tt.out, os.Stderr)
			handler := NewCopyHandler(printer)

			// Verify it's the correct concrete type
			if _, ok := handler.(*CopyHandler); !ok {
				t.Error("NewCopyHandler() does not return a *CopyHandler")
			}
		})
	}
}

func TestCopyHandler_OnTagged(t *testing.T) {
	content := []byte("test content")
	desc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    digest.FromBytes(content),
		Size:      int64(len(content)),
	}

	tests := []struct {
		name    string
		out     io.Writer
		desc    ocispec.Descriptor
		tag     string
		want    string
		wantErr bool
	}{
		{
			name:    "tags with simple tag name",
			out:     &bytes.Buffer{},
			desc:    desc,
			tag:     "latest",
			want:    "Tagged latest\n",
			wantErr: false,
		},
		{
			name:    "tags with version tag",
			out:     &bytes.Buffer{},
			desc:    desc,
			tag:     "v1.0.0",
			want:    "Tagged v1.0.0\n",
			wantErr: false,
		},
		{
			name:    "tags with complex tag name",
			out:     &bytes.Buffer{},
			desc:    desc,
			tag:     "feature-branch-123",
			want:    "Tagged feature-branch-123\n",
			wantErr: false,
		},
		{
			name:    "tags with empty tag",
			out:     &bytes.Buffer{},
			desc:    desc,
			tag:     "",
			want:    "Tagged \n",
			wantErr: false,
		},
		{
			name:    "error when writing fails",
			out:     &errorWriter{},
			desc:    desc,
			tag:     "latest",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := output.NewPrinter(tt.out, os.Stderr)
			handler := NewCopyHandler(printer).(*CopyHandler)

			err := handler.OnTagged(tt.desc, tt.tag)
			if (err != nil) != tt.wantErr {
				t.Errorf("CopyHandler.OnTagged() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if buf, ok := tt.out.(*bytes.Buffer); ok {
					got := buf.String()
					if got != tt.want {
						t.Errorf("CopyHandler.OnTagged() output = %q, want %q", got, tt.want)
					}
				}
			}
		})
	}
}

func TestCopyHandler_Render(t *testing.T) {
	content := []byte("test content")
	testDigest := digest.FromBytes(content)

	tests := []struct {
		name    string
		out     io.Writer
		desc    ocispec.Descriptor
		want    string
		wantErr bool
	}{
		{
			name: "renders digest correctly",
			out:  &bytes.Buffer{},
			desc: ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.manifest.v1+json",
				Digest:    testDigest,
				Size:      int64(len(content)),
			},
			want:    fmt.Sprintf("Digest: %s\n", testDigest),
			wantErr: false,
		},
		{
			name: "renders empty digest",
			out:  &bytes.Buffer{},
			desc: ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.manifest.v1+json",
				Digest:    "",
				Size:      0,
			},
			want:    "Digest: \n",
			wantErr: false,
		},
		{
			name: "error when writing fails",
			out:  &errorWriter{},
			desc: ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.manifest.v1+json",
				Digest:    testDigest,
				Size:      int64(len(content)),
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := output.NewPrinter(tt.out, os.Stderr)
			handler := &CopyHandler{
				printer: printer,
				desc:    tt.desc,
			}

			err := handler.Render()
			if (err != nil) != tt.wantErr {
				t.Errorf("CopyHandler.Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if buf, ok := tt.out.(*bytes.Buffer); ok {
					got := buf.String()
					if got != tt.want {
						t.Errorf("CopyHandler.Render() output = %q, want %q", got, tt.want)
					}
				}
			}
		})
	}
}

func TestCopyHandler_OnCopied(t *testing.T) {
	content := []byte("test content")
	testDesc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    digest.FromBytes(content),
		Size:      int64(len(content)),
	}

	tests := []struct {
		name    string
		out     io.Writer
		target  *option.BinaryTarget
		desc    ocispec.Descriptor
		want    string
		wantErr bool
	}{
		{
			name: "copies between registries",
			out:  &bytes.Buffer{},
			target: &option.BinaryTarget{
				From: option.Target{
					Type:         option.TargetTypeRemote,
					RawReference: "localhost:5000/source:latest",
				},
				To: option.Target{
					Type:         option.TargetTypeRemote,
					RawReference: "localhost:5000/destination:latest",
				},
			},
			desc:    testDesc,
			want:    "Copied [registry] localhost:5000/source:latest => [registry] localhost:5000/destination:latest\n",
			wantErr: false,
		},
		{
			name: "copies from registry to oci-layout",
			out:  &bytes.Buffer{},
			target: &option.BinaryTarget{
				From: option.Target{
					Type:         option.TargetTypeRemote,
					RawReference: "localhost:5000/source:v1.0.0",
				},
				To: option.Target{
					Type:         option.TargetTypeOCILayout,
					RawReference: "./oci-layout:latest",
				},
			},
			desc:    testDesc,
			want:    "Copied [registry] localhost:5000/source:v1.0.0 => [oci-layout] ./oci-layout:latest\n",
			wantErr: false,
		},
		{
			name: "copies from oci-layout to registry",
			out:  &bytes.Buffer{},
			target: &option.BinaryTarget{
				From: option.Target{
					Type:         option.TargetTypeOCILayout,
					RawReference: "./source-layout:tag1",
				},
				To: option.Target{
					Type:         option.TargetTypeRemote,
					RawReference: "ghcr.io/example/repo:tag2",
				},
			},
			desc:    testDesc,
			want:    "Copied [oci-layout] ./source-layout:tag1 => [registry] ghcr.io/example/repo:tag2\n",
			wantErr: false,
		},
		{
			name: "copies between oci-layouts",
			out:  &bytes.Buffer{},
			target: &option.BinaryTarget{
				From: option.Target{
					Type:         option.TargetTypeOCILayout,
					RawReference: "./source:v1",
				},
				To: option.Target{
					Type:         option.TargetTypeOCILayout,
					RawReference: "./dest:v2",
				},
			},
			desc:    testDesc,
			want:    "Copied [oci-layout] ./source:v1 => [oci-layout] ./dest:v2\n",
			wantErr: false,
		},
		{
			name: "error when writing fails",
			out:  &errorWriter{},
			target: &option.BinaryTarget{
				From: option.Target{
					Type:         option.TargetTypeRemote,
					RawReference: "localhost:5000/source:latest",
				},
				To: option.Target{
					Type:         option.TargetTypeRemote,
					RawReference: "localhost:5000/destination:latest",
				},
			},
			desc:    testDesc,
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := output.NewPrinter(tt.out, os.Stderr)
			handler := NewCopyHandler(printer).(*CopyHandler)

			err := handler.OnCopied(tt.target, tt.desc)
			if (err != nil) != tt.wantErr {
				t.Errorf("CopyHandler.OnCopied() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Verify that the descriptor was stored
			if !tt.wantErr {
				if handler.desc.Digest != tt.desc.Digest {
					t.Errorf("CopyHandler.OnCopied() did not store descriptor correctly, got digest %v, want %v", handler.desc.Digest, tt.desc.Digest)
				}
				if handler.desc.MediaType != tt.desc.MediaType {
					t.Errorf("CopyHandler.OnCopied() did not store descriptor correctly, got MediaType %v, want %v", handler.desc.MediaType, tt.desc.MediaType)
				}
				if handler.desc.Size != tt.desc.Size {
					t.Errorf("CopyHandler.OnCopied() did not store descriptor correctly, got Size %v, want %v", handler.desc.Size, tt.desc.Size)
				}
			}

			if !tt.wantErr {
				if buf, ok := tt.out.(*bytes.Buffer); ok {
					got := buf.String()
					if got != tt.want {
						t.Errorf("CopyHandler.OnCopied() output = %q, want %q", got, tt.want)
					}
				}
			}
		})
	}
}

func TestCopyHandler_Integration(t *testing.T) {
	// Test that shows the flow: OnCopied followed by Render
	content := []byte("integration test content")
	testDesc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    digest.FromBytes(content),
		Size:      int64(len(content)),
	}

	buf := &bytes.Buffer{}
	printer := output.NewPrinter(buf, os.Stderr)
	handler := NewCopyHandler(printer).(*CopyHandler)

	target := &option.BinaryTarget{
		From: option.Target{
			Type:         option.TargetTypeRemote,
			RawReference: "localhost:5000/src:latest",
		},
		To: option.Target{
			Type:         option.TargetTypeRemote,
			RawReference: "localhost:5000/dst:latest",
		},
	}

	// Call OnCopied first
	err := handler.OnCopied(target, testDesc)
	if err != nil {
		t.Fatalf("OnCopied() failed: %v", err)
	}

	// Then call Render
	err = handler.Render()
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}

	got := buf.String()
	expectedCopy := "Copied [registry] localhost:5000/src:latest => [registry] localhost:5000/dst:latest\n"
	expectedDigest := fmt.Sprintf("Digest: %s\n", testDesc.Digest)
	expected := expectedCopy + expectedDigest

	if got != expected {
		t.Errorf("Integration test failed.\nGot:\n%q\nWant:\n%q", got, expected)
	}
}
