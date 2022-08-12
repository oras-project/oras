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
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/internal/file"
)

func TestFile_PrepareContent(t *testing.T) {
	// generate test content
	tempDir := t.TempDir()
	dirPath := filepath.Join(tempDir, "testdir")
	if err := os.MkdirAll(dirPath, 0777); err != nil {
		t.Fatal("error calling Mkdir(), error =", err)
	}
	content := []byte("hello world!")
	fileName := "test.txt"
	path := filepath.Join(dirPath, fileName)
	if err := ioutil.WriteFile(path, content, 0444); err != nil {
		t.Fatal("error calling WriteFile(), error =", err)
	}

	blobMediaType := "application/octet-stream"
	wantDesc := ocispec.Descriptor{
		MediaType: blobMediaType,
		Digest:    digest.FromBytes(content),
		Size:      int64(len(content)),
	}

	// test PrepareContent
	gotDesc, rc, err := file.PrepareContent(path, blobMediaType)
	defer rc.Close()
	if err != nil {
		t.Fatal("PrepareContent() error=", err)
	}
	if !reflect.DeepEqual(gotDesc, wantDesc) {
		t.Errorf("PrepareContent() = %v, want %v", gotDesc, wantDesc)
	}

	// test PrepareContent with missing file name
	_, _, err = file.PrepareContent("", blobMediaType)
	expected := "missing file name"
	if err.Error() != expected {
		t.Fatalf("PrepareContent() error = %v, wantErr %v", err, expected)
	}

	// test PrepareContent with nonexistent file
	_, _, err = file.PrepareContent("nonexistent.txt", blobMediaType)
	expected = "failed to open nonexistent.txt"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("PrepareContent() error = %v, wantErr %v", err, expected)
	}
}
