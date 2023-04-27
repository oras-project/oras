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

package io_test

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	iotest "oras.land/oras/internal/io"
)

func TestReadLine(t *testing.T) {
	type args struct {
		reader io.Reader
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{"empty line", args{strings.NewReader("")}, nil},
		{"LF", args{strings.NewReader("\n")}, nil},
		{"CR", args{strings.NewReader("\r")}, []byte("")},
		{"CRLF", args{strings.NewReader("\r\n")}, []byte("")},
		{"input", args{strings.NewReader("foo")}, []byte("foo")},
		{"input ended with LF", args{strings.NewReader("foo\n")}, []byte("foo")},
		{"input ended with CR", args{strings.NewReader("foo\r")}, []byte("foo")},
		{"input ended with CRLF", args{strings.NewReader("foo\r\n")}, []byte("foo")},
		{"input contains CR", args{strings.NewReader("foo\rbar")}, []byte("foo\rbar")},
		{"input contains LF", args{strings.NewReader("foo\nbar")}, []byte("foo")},
		{"input contains CRLF", args{strings.NewReader("foo\r\nbar")}, []byte("foo")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := iotest.ReadLine(tt.args.reader)
			if err != nil {
				t.Errorf("ReadLine() error = %v", err)
				return
			}
			if left, err := io.ReadAll(tt.args.reader); err != nil {
				if err != io.EOF {
					t.Errorf("Unexpected error in reading left: %v", err)
				}
				if len(left) != 0 || strings.ContainsAny(string(left), "\r\n") {
					t.Errorf("Unexpected character left in the reader: %q", left)
				}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadLine() = %v, want %v", got, tt.want)
			}
		})
	}
}

type mockReader struct{}

func (m *mockReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("mock error")
}

func TestReadLine_err(t *testing.T) {
	got, err := iotest.ReadLine(&mockReader{})
	if err == nil {
		t.Errorf("ReadLine() = %v, want error", got)
	}
}
