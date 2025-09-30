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
	"io"
	"os"
	"testing"

	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
)

func TestNewBlobDeleteHandler(t *testing.T) {
	printer := output.NewPrinter(&bytes.Buffer{}, os.Stderr)
	target := &option.Target{
		Type:         "registry",
		RawReference: "localhost:5000/test@sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
	}

	handler := NewBlobDeleteHandler(printer, target)

	if handler == nil {
		t.Fatal("expected a non-nil handler")
	}

	// Type assertion to access internal fields
	if bdh, ok := handler.(*BlobDeleteHandler); ok {
		if bdh.printer != printer {
			t.Errorf("expected handler.printer to be %v, got %v", printer, bdh.printer)
		}
		if bdh.target != target {
			t.Errorf("expected handler.target to be %v, got %v", target, bdh.target)
		}
	} else {
		t.Error("expected handler to be of type *BlobDeleteHandler")
	}
}

func TestBlobDeleteHandler_OnBlobMissing(t *testing.T) {
	tests := []struct {
		name       string
		out        io.Writer
		target     *option.Target
		wantErr    bool
		wantOutput string
	}{
		{
			name: "blob missing with digest reference",
			out:  &bytes.Buffer{},
			target: &option.Target{
				Type:         "registry",
				RawReference: "localhost:5000/test@sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
			},
			wantErr:    false,
			wantOutput: "Missing localhost:5000/test@sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08\n",
		},
		{
			name: "blob missing with tag reference",
			out:  &bytes.Buffer{},
			target: &option.Target{
				Type:         "registry",
				RawReference: "localhost:5000/test:latest",
			},
			wantErr:    false,
			wantOutput: "Missing localhost:5000/test:latest\n",
		},
		{
			name: "blob missing with oci layout",
			out:  &bytes.Buffer{},
			target: &option.Target{
				Type:         "oci-layout",
				RawReference: "./layout@sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
			},
			wantErr:    false,
			wantOutput: "Missing ./layout@sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08\n",
		},
		{
			name: "error writer",
			out:  &errorWriter{},
			target: &option.Target{
				Type:         "registry",
				RawReference: "localhost:5000/test@sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := output.NewPrinter(tt.out, os.Stderr)
			handler := NewBlobDeleteHandler(printer, tt.target)

			err := handler.OnBlobMissing()
			if (err != nil) != tt.wantErr {
				t.Errorf("OnBlobMissing() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				got := tt.out.(*bytes.Buffer).String()
				if got != tt.wantOutput {
					t.Errorf("OnBlobMissing() output = %q, want %q", got, tt.wantOutput)
				}
			}
		})
	}
}

func TestBlobDeleteHandler_OnBlobDeleted(t *testing.T) {
	tests := []struct {
		name       string
		out        io.Writer
		target     *option.Target
		wantErr    bool
		wantOutput string
	}{
		{
			name: "blob deleted with digest reference",
			out:  &bytes.Buffer{},
			target: &option.Target{
				Type:         "registry",
				RawReference: "localhost:5000/test@sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
			},
			wantErr:    false,
			wantOutput: "Deleted [registry] localhost:5000/test@sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08\n",
		},
		{
			name: "blob deleted with tag reference",
			out:  &bytes.Buffer{},
			target: &option.Target{
				Type:         "registry",
				RawReference: "localhost:5000/test:latest",
			},
			wantErr:    false,
			wantOutput: "Deleted [registry] localhost:5000/test:latest\n",
		},
		{
			name: "blob deleted with oci layout",
			out:  &bytes.Buffer{},
			target: &option.Target{
				Type:         "oci-layout",
				RawReference: "./layout@sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
			},
			wantErr:    false,
			wantOutput: "Deleted [oci-layout] ./layout@sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08\n",
		},
		{
			name: "error writer",
			out:  &errorWriter{},
			target: &option.Target{
				Type:         "registry",
				RawReference: "localhost:5000/test@sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := output.NewPrinter(tt.out, os.Stderr)
			handler := NewBlobDeleteHandler(printer, tt.target)

			err := handler.OnBlobDeleted()
			if (err != nil) != tt.wantErr {
				t.Errorf("OnBlobDeleted() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				got := tt.out.(*bytes.Buffer).String()
				if got != tt.wantOutput {
					t.Errorf("OnBlobDeleted() output = %q, want %q", got, tt.wantOutput)
				}
			}
		})
	}
}
