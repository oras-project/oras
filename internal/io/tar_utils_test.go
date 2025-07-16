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
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	iotest "oras.land/oras/internal/io"
)

func TestTarDirectory(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir, err := os.MkdirTemp("", "tar-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files and directories
	testFiles := map[string]string{
		"file1.txt":        "content of file 1",
		"file2.txt":        "content of file 2",
		"subdir/file3.txt": "content of file 3",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", path, err)
		}
	}

	// Create a buffer to store the tar archive
	var buf bytes.Buffer

	// Call TarDirectory
	err = iotest.TarDirectory(context.Background(), &buf, tmpDir)
	if err != nil {
		t.Fatalf("TarDirectory failed: %v", err)
	}

	// Read and verify the tar archive
	tr := tar.NewReader(&buf)
	foundFiles := make(map[string]string)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to read tar header: %v", err)
		}

		if header.FileInfo().IsDir() {
			continue
		}

		var contentBuf bytes.Buffer
		if _, err := io.Copy(&contentBuf, tr); err != nil {
			t.Fatalf("Failed to read file content: %v", err)
		}

		foundFiles[header.Name] = contentBuf.String()
	}

	// Verify all files are present with correct content
	for path, expectedContent := range testFiles {
		path = filepath.ToSlash(path) // Convert to forward slashes for consistency
		content, found := foundFiles[path]
		if !found {
			t.Errorf("File %s not found in tar archive", path)
			continue
		}

		if content != expectedContent {
			t.Errorf("File %s has wrong content: got %q, want %q", path, content, expectedContent)
		}
	}

	// Verify no extra files were included
	for path := range foundFiles {
		_, expected := testFiles[path]
		if !expected {
			t.Errorf("Unexpected file in tar archive: %s", path)
		}
	}
}

func TestTarDirectory_InvalidSource(t *testing.T) {
	var buf bytes.Buffer

	// Test with non-existent directory
	err := iotest.TarDirectory(context.Background(), &buf, "/path/does/not/exist")
	if err == nil {
		t.Error("Expected error for non-existent source directory, but got nil")
	}

	// Create a temporary file to test with non-directory source
	tmpFile, err := os.CreateTemp("", "not-a-dir")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	err = iotest.TarDirectory(context.Background(), &buf, tmpFile.Name())
	if err == nil {
		t.Error("Expected error for file as source, but got nil")
	}
}

func TestTarDirectory_ContextCancellation(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "tar-test-cancel")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file in the directory
	if err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	var buf bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context immediately to simulate interruption
	cancel()

	err = iotest.TarDirectory(ctx, &buf, tmpDir)
	if err == nil {
		t.Error("Expected context cancellation error, but got nil")
	}
}
