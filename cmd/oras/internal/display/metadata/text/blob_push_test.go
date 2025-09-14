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
	"reflect"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
)

func TestNewBlobPushHandler(t *testing.T) {
	printer := output.NewPrinter(&bytes.Buffer{}, os.Stderr)
	desc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.layer.v1.tar",
		Digest:    digest.FromString("test"),
		Size:      123,
	}

	handler := NewBlobPushHandler(printer, desc)

	if handler == nil {
		t.Fatal("expected a non-nil handler")
	}

	// Type assertion to access internal fields
	if bph, ok := handler.(*BlobPushHandler); ok {
		if bph.printer != printer {
			t.Errorf("expected handler.printer to be %v, got %v", printer, bph.printer)
		}
		if !reflect.DeepEqual(bph.desc, desc) {
			t.Errorf("expected handler.desc to be %v, got %v", desc, bph.desc)
		}
	} else {
		t.Error("expected handler to be of type *BlobPushHandler")
	}
}

func TestBlobPushHandler_OnBlobPushed(t *testing.T) {
	tests := []struct {
		name       string
		out        io.Writer
		target     *option.Target
		wantErr    bool
		wantOutput string
	}{
		{
			name: "successful push output",
			out:  &bytes.Buffer{},
			target: &option.Target{
				Type:         "registry",
				RawReference: "localhost:5000/test:latest",
			},
			wantErr:    false,
			wantOutput: "Pushed: [registry] localhost:5000/test:latest\n",
		},
		{
			name: "oci layout target",
			out:  &bytes.Buffer{},
			target: &option.Target{
				Type:         "oci-layout",
				RawReference: "./layout:latest",
			},
			wantErr:    false,
			wantOutput: "Pushed: [oci-layout] ./layout:latest\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := output.NewPrinter(tt.out, os.Stderr)
			desc := ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.layer.v1.tar",
				Digest:    digest.FromString("test"),
				Size:      123,
			}
			handler := NewBlobPushHandler(printer, desc)

			err := handler.OnBlobPushed(tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnBlobPushed() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				got := tt.out.(*bytes.Buffer).String()
				if got != tt.wantOutput {
					t.Errorf("OnBlobPushed() output = %q, want %q", got, tt.wantOutput)
				}
			}
		})
	}
}

func TestBlobPushHandler_Render(t *testing.T) {
	content := []byte("test content")
	testDigest := digest.FromBytes(content)

	tests := []struct {
		name       string
		out        io.Writer
		desc       ocispec.Descriptor
		wantErr    bool
		wantOutput string
	}{
		{
			name: "render digest output",
			out:  &bytes.Buffer{},
			desc: ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.layer.v1.tar",
				Digest:    testDigest,
				Size:      int64(len(content)),
			},
			wantErr:    false,
			wantOutput: fmt.Sprintf("Digest: %s\n", testDigest),
		},
		{
			name: "render with different media type",
			out:  &bytes.Buffer{},
			desc: ocispec.Descriptor{
				MediaType: "application/vnd.example.file",
				Digest:    testDigest,
				Size:      int64(len(content)),
			},
			wantErr:    false,
			wantOutput: fmt.Sprintf("Digest: %s\n", testDigest),
		},
		{
			name: "error writer",
			out:  &errorWriter{},
			desc: ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.layer.v1.tar",
				Digest:    testDigest,
				Size:      int64(len(content)),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := output.NewPrinter(tt.out, os.Stderr)
			handler := NewBlobPushHandler(printer, tt.desc)

			err := handler.Render()
			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				got := tt.out.(*bytes.Buffer).String()
				if got != tt.wantOutput {
					t.Errorf("Render() output = %q, want %q", got, tt.wantOutput)
				}
			}
		})
	}
}
