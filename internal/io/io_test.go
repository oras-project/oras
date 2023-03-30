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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	iotest "oras.land/oras/internal/io"
)

const manifestMediaType = "application/vnd.oci.image.manifest.v1+json"
const blobMediaType = "application/mock-octet-stream"
const manifest = `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"mediaType":"application/vnd.unknown.config.v1+json","digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2},"layers":[{"mediaType":"application/vnd.oci.image.layer.v1.tar","digest":"sha256:5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03","size":6,"annotations":{"org.opencontainers.image.title":"hello.txt"}}]}`

func TestFile_PrepareManifestContent(t *testing.T) {
	// generate test content
	tempDir := t.TempDir()
	content := []byte(manifest)
	fileName := "manifest.json"
	path := filepath.Join(tempDir, fileName)
	if err := os.WriteFile(path, content, 0444); err != nil {
		t.Fatal("error calling WriteFile(), error =", err)
	}

	want := []byte(manifest)

	// test PrepareManifestContent
	got, err := iotest.PrepareManifestContent(path)
	if err != nil {
		t.Fatal("PrepareManifestContent() error=", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("PrepareManifestContent() = %v, want %v", got, want)
	}
}

func TestFile_PrepareManifestContent_fromStdin(t *testing.T) {
	// generate test content
	content := []byte(manifest)
	tempDir := t.TempDir()
	fileName := "manifest.json"
	path := filepath.Join(tempDir, fileName)
	tmpfile, err := os.Create(path)
	if err != nil {
		t.Fatal("error calling os.Create(), error =", err)
	}
	defer tmpfile.Close()

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal("error calling Write(), error =", err)
	}
	if _, err := tmpfile.Seek(0, 0); err != nil {
		t.Fatal("error calling Seek(), error =", err)
	}

	os.Stdin = tmpfile
	defer func(stdin *os.File) { os.Stdin = stdin }(os.Stdin)

	want := []byte(manifest)

	// test PrepareManifestContent read from stdin
	got, err := iotest.PrepareManifestContent("-")
	if err != nil {
		t.Fatal("PrepareManifestContent() error=", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("PrepareManifestContent() = %v, want %v", got, want)
	}
}

func TestFile_PrepareManifestContent_errMissingFileName(t *testing.T) {
	// test PrepareManifestContent with missing file name
	_, err := iotest.PrepareManifestContent("")
	expected := "missing file name"
	if err.Error() != expected {
		t.Fatalf("PrepareManifestContent() error = %v, wantErr %v", err, expected)
	}
}

func TestFile_PrepareManifestContent_errReadFile(t *testing.T) {
	// test PrepareManifestContent with nonexistent file
	_, err := iotest.PrepareManifestContent("nonexistent.txt")
	expected := "failed to read nonexistent.txt"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("PrepareManifestContent() error = %v, wantErr %v", err, expected)
	}
}

func TestFile_PrepareBlobContent(t *testing.T) {
	// generate test content
	tempDir := t.TempDir()
	content := []byte("hello world!")
	fileName := "test.txt"
	path := filepath.Join(tempDir, fileName)
	if err := os.WriteFile(path, content, 0444); err != nil {
		t.Fatal("error calling WriteFile(), error =", err)
	}

	want := ocispec.Descriptor{
		MediaType: blobMediaType,
		Digest:    digest.FromBytes(content),
		Size:      int64(len(content)),
	}

	// test PrepareBlobContent
	got, rc, err := iotest.PrepareBlobContent(path, blobMediaType, "", -1)
	if err != nil {
		t.Fatal("PrepareBlobContent() error=", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("PrepareBlobContent() = %v, want %v", got, want)
	}
	actualContent, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal("PrepareBlobContent(): not able to read content from rc, error=", err)
	}
	err = rc.Close()
	if err != nil {
		t.Fatal("error calling rc.Close(), error =", err)
	}
	if !reflect.DeepEqual(actualContent, content) {
		t.Errorf("PrepareBlobContent() = %v, want %v", actualContent, content)
	}

	// test PrepareBlobContent with provided digest and size
	dgstStr := "sha256:9a201d228ebd966211f7d1131be19f152be428bd373a92071c71d8deaf83b3e5"
	size := int64(12)
	got, rc, err = iotest.PrepareBlobContent(path, blobMediaType, dgstStr, size)
	if err != nil {
		t.Fatal("PrepareBlobContent() error=", err)
	}
	want = ocispec.Descriptor{
		MediaType: blobMediaType,
		Digest:    digest.Digest(dgstStr),
		Size:      size,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("PrepareBlobContent() = %v, want %v", got, want)
	}
	actualContent, err = io.ReadAll(rc)
	if err != nil {
		t.Fatal("PrepareBlobContent(): not able to read content from rc, error=", err)
	}
	err = rc.Close()
	if err != nil {
		t.Fatal("error calling rc.Close(), error =", err)
	}
	if !reflect.DeepEqual(actualContent, content) {
		t.Errorf("PrepareBlobContent() = %v, want %v", actualContent, content)
	}

	// test PrepareBlobContent with provided size, but the size does not match the
	// actual content size
	_, _, err = iotest.PrepareBlobContent(path, blobMediaType, "", 15)
	expected := fmt.Sprintf("input size %d does not match the actual content size %d", 15, size)
	if err.Error() != expected {
		t.Fatalf("PrepareBlobContent() error = %v, wantErr %v", err, expected)
	}
}

func TestFile_PrepareBlobContent_fromStdin(t *testing.T) {
	// generate test content
	content := []byte("hello world!")
	tempDir := t.TempDir()
	fileName := "test.txt"
	path := filepath.Join(tempDir, fileName)
	tmpfile, err := os.Create(path)
	if err != nil {
		t.Fatal("error calling os.Create(), error =", err)
	}
	defer tmpfile.Close()

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal("error calling Write(), error =", err)
	}
	if _, err := tmpfile.Seek(0, 0); err != nil {
		t.Fatal("error calling Seek(), error =", err)
	}

	defer func(stdin *os.File) { os.Stdin = stdin }(os.Stdin)

	os.Stdin = tmpfile
	dgst := digest.FromBytes(content)
	size := int64(len(content))
	wantDesc := ocispec.Descriptor{
		MediaType: blobMediaType,
		Digest:    digest.FromBytes(content),
		Size:      int64(len(content)),
	}

	// test PrepareBlobContent with provided digest and size
	gotDesc, gotRc, err := iotest.PrepareBlobContent("-", blobMediaType, string(dgst), size)
	defer gotRc.Close()
	if err != nil {
		t.Fatal("PrepareBlobContent() error=", err)
	}
	if !reflect.DeepEqual(gotDesc, wantDesc) {
		t.Errorf("PrepareBlobContent() = %v, want %v", gotDesc, wantDesc)
	}
	if _, err = tmpfile.Seek(0, io.SeekStart); err != nil {
		t.Fatal("error calling Seek(), error =", err)
	}
	if !reflect.DeepEqual(gotRc, tmpfile) {
		t.Errorf("PrepareBlobContent() = %v, want %v", gotRc, tmpfile)
	}

	// test PrepareBlobContent from stdin with missing size
	_, _, err = iotest.PrepareBlobContent("-", blobMediaType, "", -1)
	expected := "content size must be provided if it is read from stdin"
	if err.Error() != expected {
		t.Fatalf("PrepareBlobContent() error = %v, wantErr %v", err, expected)
	}

	// test PrepareBlobContent from stdin with missing digest
	_, _, err = iotest.PrepareBlobContent("-", blobMediaType, "", 5)
	expected = "content digest must be provided if it is read from stdin"
	if err.Error() != expected {
		t.Fatalf("PrepareBlobContent() error = %v, wantErr %v", err, expected)
	}
}

func TestFile_PrepareBlobContent_errDigestInvalidFormat(t *testing.T) {
	// test PrepareBlobContent from stdin with invalid digest
	invalidDgst := "xyz"
	_, _, err := iotest.PrepareBlobContent("-", blobMediaType, invalidDgst, 12)
	if !errors.Is(err, digest.ErrDigestInvalidFormat) {
		t.Fatalf("PrepareBlobContent() error = %v, wantErr %v", err, digest.ErrDigestInvalidFormat)
	}
}

func TestFile_PrepareBlobContent_errMissingFileName(t *testing.T) {
	// test PrepareBlobContent with missing file name
	_, _, err := iotest.PrepareBlobContent("", blobMediaType, "", -1)
	expected := "missing file name"
	if err.Error() != expected {
		t.Fatalf("PrepareBlobContent() error = %v, wantErr %v", err, expected)
	}
}

func TestFile_PrepareBlobContent_errOpenFile(t *testing.T) {
	// test PrepareBlobContent with nonexistent file
	_, _, err := iotest.PrepareBlobContent("nonexistent.txt", blobMediaType, "", -1)
	expected := "failed to open nonexistent.txt"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("PrepareBlobContent() error = %v, wantErr %v", err, expected)
	}
}

func TestFile_ParseMediaType(t *testing.T) {
	// generate test content
	content := []byte(manifest)

	// test ParseMediaType
	want := manifestMediaType
	got, err := iotest.ParseMediaType(content)
	if err != nil {
		t.Fatal("ParseMediaType() error=", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseMediaType() = %v, want %v", got, want)
	}
}

func TestFile_ParseMediaType_invalidContent_notAJson(t *testing.T) {
	// generate test content
	content := []byte("manifest")

	// test ParseMediaType
	_, err := iotest.ParseMediaType(content)
	expected := "not a valid json file"
	if err.Error() != expected {
		t.Fatalf("ParseMediaType() error = %v, wantErr %v", err, expected)
	}
}

func TestFile_ParseMediaType_invalidContent_missingMediaType(t *testing.T) {
	// generate test content
	content := []byte(`{"schemaVersion":2}`)

	// test ParseMediaType
	_, err := iotest.ParseMediaType(content)
	expected := "media type is not recognized"
	if err.Error() != expected {
		t.Fatalf("ParseMediaType() error = %v, wantErr %v", err, expected)
	}
}

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
