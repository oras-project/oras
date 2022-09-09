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

package file_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/internal/file"
)

const blobMediaType = "application/mock-octet-stream"

func TestFile_PrepareContent(t *testing.T) {
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

	// test PrepareContent
	got, rc, err := file.PrepareContent(path, blobMediaType, "", -1)
	if err != nil {
		t.Fatal("PrepareContent() error=", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("PrepareContent() = %v, want %v", got, want)
	}
	actualContent, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal("PrepareContent(): not able to read content from rc, error=", err)
	}
	err = rc.Close()
	if err != nil {
		t.Fatal("error calling rc.Close(), error =", err)
	}
	if !reflect.DeepEqual(actualContent, content) {
		t.Errorf("PrepareContent() = %v, want %v", actualContent, content)
	}

	// test PrepareContent with provided digest and size
	dgstStr := "sha256:9a201d228ebd966211f7d1131be19f152be428bd373a92071c71d8deaf83b3e5"
	size := int64(12)
	got, rc, err = file.PrepareContent(path, blobMediaType, dgstStr, size)
	if err != nil {
		t.Fatal("PrepareContent() error=", err)
	}
	want = ocispec.Descriptor{
		MediaType: blobMediaType,
		Digest:    digest.Digest(dgstStr),
		Size:      size,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("PrepareContent() = %v, want %v", got, want)
	}
	actualContent, err = io.ReadAll(rc)
	if err != nil {
		t.Fatal("PrepareContent(): not able to read content from rc, error=", err)
	}
	err = rc.Close()
	if err != nil {
		t.Fatal("error calling rc.Close(), error =", err)
	}
	if !reflect.DeepEqual(actualContent, content) {
		t.Errorf("PrepareContent() = %v, want %v", actualContent, content)
	}
}

func TestFile_PrepareContent_fromStdin(t *testing.T) {
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

	// test PrepareContent with provided digest and size
	gotDesc, gotRc, err := file.PrepareContent("-", blobMediaType, string(dgst), size)
	defer gotRc.Close()
	if err != nil {
		t.Fatal("PrepareContent() error=", err)
	}
	if !reflect.DeepEqual(gotDesc, wantDesc) {
		t.Errorf("PrepareContent() = %v, want %v", gotDesc, wantDesc)
	}
	if _, err = tmpfile.Seek(0, io.SeekStart); err != nil {
		t.Fatal("error calling Seek(), error =", err)
	}
	if !reflect.DeepEqual(gotRc, tmpfile) {
		t.Errorf("PrepareContent() = %v, want %v", gotRc, tmpfile)
	}

	// test PrepareContent from stdin with missing digest and size
	_, _, err = file.PrepareContent("-", blobMediaType, "", -1)
	expected := "content size and digest must be provided if it is read from stdin"
	if err.Error() != expected {
		t.Fatalf("PrepareContent() error = %v, wantErr %v", err, expected)
	}
}

func TestFile_PrepareContent_errDigestInvalidFormat(t *testing.T) {
	// test PrepareContent from stdin with invalid digest
	invalidDgst := "xyz"
	_, _, err := file.PrepareContent("-", blobMediaType, invalidDgst, 12)
	if !errors.Is(err, digest.ErrDigestInvalidFormat) {
		t.Fatalf("PrepareContent() error = %v, wantErr %v", err, digest.ErrDigestInvalidFormat)
	}
}

func TestFile_PrepareContent_errMissingFileName(t *testing.T) {
	// test PrepareContent with missing file name
	_, _, err := file.PrepareContent("", blobMediaType, "", -1)
	expected := "missing file name"
	if err.Error() != expected {
		t.Fatalf("PrepareContent() error = %v, wantErr %v", err, expected)
	}
}

func TestFile_PrepareContent_errOpenFile(t *testing.T) {
	// test PrepareContent with nonexistent file
	_, _, err := file.PrepareContent("nonexistent.txt", blobMediaType, "", -1)
	expected := "failed to open nonexistent.txt"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("PrepareContent() error = %v, wantErr %v", err, expected)
	}
}
