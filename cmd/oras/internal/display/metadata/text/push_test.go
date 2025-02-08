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
	"oras.land/oras/cmd/oras/internal/output"
)

type errorWriter struct{}

// Write implements the io.Writer interface and returns an error in Write.
func (w *errorWriter) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("got an error")
}

func TestPushHandler_Render(t *testing.T) {
	content := []byte("content")
	tests := []struct {
		name    string
		out     io.Writer
		root    ocispec.Descriptor
		wantErr bool
	}{
		{
			"good path",
			&bytes.Buffer{},
			ocispec.Descriptor{
				MediaType: "example",
				Digest:    digest.FromBytes(content),
				Size:      int64(len(content)),
			},
			false,
		},
		{
			"error path",
			&errorWriter{},
			ocispec.Descriptor{
				MediaType: "example",
				Digest:    digest.FromBytes(content),
				Size:      int64(len(content)),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := output.NewPrinter(tt.out, os.Stderr)
			p := &PushHandler{
				printer: printer,
				root:    tt.root,
			}
			if err := p.Render(); (err != nil) != tt.wantErr {
				t.Errorf("PushHandler.Render() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
