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
	got, rc, err := file.PrepareContent(path, blobMediaType)
	defer rc.Close()
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
	if !reflect.DeepEqual(actualContent, content) {
		t.Errorf("PrepareContent() = %v, want %v", actualContent, content)
	}
}

func TestFile_PrepareContent_errMissingFileName(t *testing.T) {
	// test PrepareContent with missing file name
	_, _, err := file.PrepareContent("", blobMediaType)
	expected := "missing file name"
	if err.Error() != expected {
		t.Fatalf("PrepareContent() error = %v, wantErr %v", err, expected)
	}
}

func TestFile_PrepareContent_errOpenFile(t *testing.T) {
	// test PrepareContent with nonexistent file
	_, _, err := file.PrepareContent("nonexistent.txt", blobMediaType)
	expected := "failed to open nonexistent.txt"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("PrepareContent() error = %v, wantErr %v", err, expected)
	}
}
