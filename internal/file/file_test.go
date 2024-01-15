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
	"fmt"
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
	got, err := file.PrepareManifestContent(path)
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
	got, err := file.PrepareManifestContent("-")
	if err != nil {
		t.Fatal("PrepareManifestContent() error=", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("PrepareManifestContent() = %v, want %v", got, want)
	}
}

func TestFile_PrepareManifestContent_errMissingFileName(t *testing.T) {
	// test PrepareManifestContent with missing file name
	_, err := file.PrepareManifestContent("")
	expected := "missing file name"
	if err.Error() != expected {
		t.Fatalf("PrepareManifestContent() error = %v, wantErr %v", err, expected)
	}
}

func TestFile_PrepareManifestContent_errReadFile(t *testing.T) {
	// test PrepareManifestContent with nonexistent file
	_, err := file.PrepareManifestContent("nonexistent.txt")
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
	got, rc, err := file.PrepareBlobContent(path, blobMediaType, "", -1)
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
	got, rc, err = file.PrepareBlobContent(path, blobMediaType, dgstStr, size)
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
	_, _, err = file.PrepareBlobContent(path, blobMediaType, "", 15)
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
	gotDesc, gotRc, err := file.PrepareBlobContent("-", blobMediaType, string(dgst), size)
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
	_, _, err = file.PrepareBlobContent("-", blobMediaType, "", -1)
	expected := "content size must be provided if it is read from stdin"
	if err.Error() != expected {
		t.Fatalf("PrepareBlobContent() error = %v, wantErr %v", err, expected)
	}

	// test PrepareBlobContent from stdin with missing digest
	_, _, err = file.PrepareBlobContent("-", blobMediaType, "", 5)
	expected = "content digest must be provided if it is read from stdin"
	if err.Error() != expected {
		t.Fatalf("PrepareBlobContent() error = %v, wantErr %v", err, expected)
	}
}

func TestFile_PrepareBlobContent_errDigestInvalidFormat(t *testing.T) {
	// test PrepareBlobContent from stdin with invalid digest
	invalidDgst := "xyz"
	_, _, err := file.PrepareBlobContent("-", blobMediaType, invalidDgst, 12)
	if !errors.Is(err, digest.ErrDigestInvalidFormat) {
		t.Fatalf("PrepareBlobContent() error = %v, wantErr %v", err, digest.ErrDigestInvalidFormat)
	}
}

func TestFile_PrepareBlobContent_errMissingFileName(t *testing.T) {
	// test PrepareBlobContent with missing file name
	_, _, err := file.PrepareBlobContent("", blobMediaType, "", -1)
	expected := "missing file name"
	if err.Error() != expected {
		t.Fatalf("PrepareBlobContent() error = %v, wantErr %v", err, expected)
	}
}

func TestFile_PrepareBlobContent_errOpenFile(t *testing.T) {
	// test PrepareBlobContent with nonexistent file
	_, _, err := file.PrepareBlobContent("nonexistent.txt", blobMediaType, "", -1)
	expected := "failed to open nonexistent.txt"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("PrepareBlobContent() error = %v, wantErr %v", err, expected)
	}
}
