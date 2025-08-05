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
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temporary directory: %v", err)
		}
	}()

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

	// Note: We're not testing symlinks as they are intentionally not supported
	// by the TarDirectory function design

	// Call TarDirectory
	err = iotest.TarDirectory(&buf, tmpDir)
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
	t.Run("Source directory does not exist", func(t *testing.T) {
		err := iotest.TarDirectory(&buf, "/path/does/not/exist")
		if err == nil {
			t.Error("Expected error for non-existent source directory, but got nil")
		}
	})

	// Create a temporary file to test with non-directory source
	t.Run("Source is not a directory", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "not-a-dir")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer func() {
			if err := os.Remove(tmpFile.Name()); err != nil {
				t.Logf("Failed to remove temporary file: %v", err)
			}
		}()
		defer func() {
			if err := tmpFile.Close(); err != nil {
				t.Logf("Failed to close temporary file: %v", err)
			}
		}()

		err = iotest.TarDirectory(&buf, tmpFile.Name())
		if err == nil {
			t.Error("Expected error for file as source, but got nil")
		}
	})
}

func TestIsTarFile(t *testing.T) {
	// Test case 1: File with .tar extension
	t.Run("File with .tar extension", func(t *testing.T) {
		tmpDir := t.TempDir()
		tarFilePath := filepath.Join(tmpDir, "test.tar")
		if err := os.WriteFile(tarFilePath, []byte("not really a tar file"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		isTar, err := iotest.IsTarFile(tarFilePath)
		if err != nil {
			t.Errorf("IsTarFile returned unexpected error: %v", err)
		}
		if !isTar {
			t.Error("IsTarFile should identify .tar files as tar archives regardless of content")
		}
	})

	// Test case 2: File with .taR extension (case-insensitive)
	t.Run("File with .taR extension (mixed case)", func(t *testing.T) {
		tmpDir := t.TempDir()
		tarFilePath := filepath.Join(tmpDir, "test.taR")
		if err := os.WriteFile(tarFilePath, []byte("not really a tar file"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		isTar, err := iotest.IsTarFile(tarFilePath)
		if err != nil {
			t.Errorf("IsTarFile returned unexpected error: %v", err)
		}
		if !isTar {
			t.Error("IsTarFile should identify .TAR files as tar archives (case-insensitive)")
		}
	})

	// Test case 3: File with tar magic number but without .tar extension
	t.Run("File with tar magic number", func(t *testing.T) {
		tmpDir := t.TempDir()
		nonTarPath := filepath.Join(tmpDir, "test.bin")

		// Create a file with the tar magic number at position 257
		content := make([]byte, 512) // Typical tar header size
		copy(content[257:], []byte("ustar"))

		if err := os.WriteFile(nonTarPath, content, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		isTar, err := iotest.IsTarFile(nonTarPath)
		if err != nil {
			t.Errorf("IsTarFile returned unexpected error: %v", err)
		}
		if !isTar {
			t.Error("IsTarFile should identify files with tar magic number as tar archives")
		}
	})

	// Test case 4: Regular file without tar magic number
	t.Run("Regular non-tar file", func(t *testing.T) {
		tmpDir := t.TempDir()
		regularFilePath := filepath.Join(tmpDir, "regular.txt")

		// Create a file that's large enough but doesn't have the tar magic number
		content := make([]byte, 512)
		copy(content[257:], []byte("nope!"))

		if err := os.WriteFile(regularFilePath, content, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		isTar, err := iotest.IsTarFile(regularFilePath)
		if err != nil {
			t.Errorf("IsTarFile returned unexpected error: %v", err)
		}
		if isTar {
			t.Error("IsTarFile should not identify regular files as tar archives")
		}
	})

	// Test case 5: Non-existent file
	t.Run("Non-existent file", func(t *testing.T) {
		tmpDir := t.TempDir()
		nonExistentPath := filepath.Join(tmpDir, "does-not-exist.txt")

		isTar, err := iotest.IsTarFile(nonExistentPath)
		if err == nil {
			t.Error("IsTarFile should return error for non-existent file")
		}
		if isTar {
			t.Error("IsTarFile should return false for non-existent file")
		}
	})

	// Test case 6: File too small for magic number check
	t.Run("File too small for magic number", func(t *testing.T) {
		tmpDir := t.TempDir()
		smallFilePath := filepath.Join(tmpDir, "small.txt")

		if err := os.WriteFile(smallFilePath, []byte("small file"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		isTar, err := iotest.IsTarFile(smallFilePath)
		if err == nil {
			t.Error("IsTarFile should return error for file too small for magic number check")
		}
		if isTar {
			t.Error("IsTarFile should return false for file too small for magic number check")
		}
	})
}
