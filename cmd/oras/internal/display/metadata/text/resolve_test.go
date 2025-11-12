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
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/output"
)

func TestResolveHandle_onResolved(t *testing.T) {
	var buf bytes.Buffer
	printer := output.NewPrinter(&buf, &buf)

	tests := []struct {
		name     string
		fullRef  bool
		path     string
		expected string
	}{
		{
			name:     "full reference output",
			fullRef:  true,
			path:     "localhost:5000/test",
			expected: "localhost:5000/test@sha256:abcd1234\n",
		},
		{
			name:     "digest only output",
			fullRef:  false,
			path:     "localhost:5000/test",
			expected: "sha256:abcd1234\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			handler := NewResolveHandler(printer, tt.fullRef, tt.path)

			desc := ocispec.Descriptor{
				Digest: "sha256:abcd1234",
			}

			err := handler.OnResolved(desc)
			if err != nil {
				t.Fatalf("OnResolved() error = %v", err)
			}

			got := buf.String()
			if got != tt.expected {
				t.Errorf("OnResolved() output = %q, want %q", got, tt.expected)
			}
		})
	}
}
