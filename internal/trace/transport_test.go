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

package trace

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func Test_isPrintableContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		want        bool
	}{
		{
			name:        "Empty content type",
			contentType: "",
			want:        false,
		},
		{
			name:        "General JSON type",
			contentType: "application/json",
			want:        true,
		},
		{
			name:        "Manifest type in JSON",
			contentType: "application/vnd.oci.image.manifest.v1+json",
			want:        true,
		},
		{
			name:        "Random content type in JSON",
			contentType: "application/whatever+json",
			want:        true,
		},
		{
			name:        "Plain text type",
			contentType: "text/plain",
			want:        true,
		},
		{
			name:        "Plain text type with charset",
			contentType: "text/plain; charset=utf-8",
			want:        true,
		},
		{
			name:        "HTML text type",
			contentType: "text/html",
			want:        true,
		},
		{
			name:        "HTML text type with charset",
			contentType: "text/html; charset=utf-8",
			want:        true,
		},
		{
			name:        "Binary type",
			contentType: "application/octet-stream",
			want:        false,
		},
		{
			name:        "Unknown type",
			contentType: "unknown/unknown",
			want:        false,
		},
		{
			name:        "Invalid type",
			contentType: "application/",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPrintableContentType(tt.contentType); got != tt.want {
				t.Errorf("isPrintableContentType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_logResponseBody(t *testing.T) {
	tests := []struct {
		name string
		resp *http.Response
		want string
	}{
		{
			name: "Nil body",
			resp: &http.Response{
				Body: nil,
			},
			want: "",
		},
		{
			name: "No body",
			resp: &http.Response{
				Body: http.NoBody,
			},
			want: "",
		},
		{
			name: "Empty body",
			resp: &http.Response{
				Body:          io.NopCloser(strings.NewReader("")),
				ContentLength: 0,
			},
			want: "",
		},
		{
			name: "Unknown content length",
			resp: &http.Response{
				Body:          io.NopCloser(strings.NewReader("whatever")),
				ContentLength: -1,
			},
			want: "",
		},
		{
			name: "Non-printable content type",
			resp: &http.Response{
				Body:          io.NopCloser(strings.NewReader("binary data")),
				ContentLength: 10,
				Header:        http.Header{"Content-Type": []string{"application/octet-stream"}},
			},
			want: "   Body of content type \"application/octet-stream\" is not printed",
		},
		{
			name: "Body larger than limit",
			resp: &http.Response{
				Body:          io.NopCloser(strings.NewReader(strings.Repeat("a", int(payloadSizeLimit+1)))),
				ContentLength: payloadSizeLimit + 1,
				Header:        http.Header{"Content-Type": []string{"text/plain"}},
			},
			want: fmt.Sprintf("   Body larger than %d bytes is not printed", payloadSizeLimit),
		},
		{
			name: "Printable content type within limit",
			resp: &http.Response{
				Body:          io.NopCloser(strings.NewReader("printable data")),
				ContentLength: int64(len("printable data")),
				Header:        http.Header{"Content-Type": []string{"text/plain"}},
			},
			want: "printable data",
		},
		{
			name: "Actual body size is larger than content length",
			resp: &http.Response{
				Body:          io.NopCloser(strings.NewReader("data")),
				ContentLength: 3,
				Header:        http.Header{"Content-Type": []string{"text/plain"}},
			},
			want: "dat",
		},
		{
			name: "Actual body size is smaller than content length",
			resp: &http.Response{
				Body:          io.NopCloser(strings.NewReader("data")),
				ContentLength: 5,
				Header:        http.Header{"Content-Type": []string{"text/plain"}},
			},
			want: "data",
		},
		{
			name: "Error reading body",
			resp: &http.Response{
				Body:          io.NopCloser(&errorReader{}),
				ContentLength: 10,
				Header:        http.Header{"Content-Type": []string{"text/plain"}},
			},
			want: "   Error reading response body: mock error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := logResponseBody(tt.resp); got != tt.want {
				t.Errorf("logResponseBody() = %v, want %v", got, tt.want)
			}
		})
	}
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("mock error")
}
