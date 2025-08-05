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

package io

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// TarDirectory creates a tar archive from the contents of sourceDir and writes it to the given writer.
func TarDirectory(writer io.Writer, sourceDir string) (tarErr error) {
	// Ensure sourceDir exists and is a directory
	fi, err := os.Stat(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to stat source directory: %w", err)
	}
	if !fi.IsDir() {
		return fmt.Errorf("source is not a directory: %s", sourceDir)
	}

	// Create a new tar writer
	tw := tar.NewWriter(writer)
	defer func() {
		closeErr := tw.Close()
		if tarErr == nil {
			tarErr = closeErr
		}
	}()

	return tw.AddFS(os.DirFS(sourceDir))
}

// IsTarFile loosely checks whether the given file path refers to a tar archive
// by examining its extension and magic number.
func IsTarFile(path string) (bool, error) {
	// loose check: consider *.tar files as tar archives
	if strings.EqualFold(filepath.Ext(path), ".tar") {
		return true, nil
	}

	// check the magic number to determine the file type
	fp, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("failed to open file %q: %w", path, err)
	}
	defer func() {
		_ = fp.Close()
	}()

	// read 5 bytes ("ustar") at the position where the magic number is located
	magic := make([]byte, 5)
	_, err = fp.ReadAt(magic, 257) // ustar magic number starts at byte 257
	if err != nil {
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		return false, fmt.Errorf("failed to read magic number from file %q: %w", path, err)
	}
	return bytes.Equal(magic, []byte("ustar")), nil
}
