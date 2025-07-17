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
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// TarDirectory creates a tar archive from the specified directory.
// The sourceDir is the root directory to start archiving from.
// All files and directories within sourceDir will be added to the tar archive.
// The paths in the archive will be relative to sourceDir.
func TarDirectory(ctx context.Context, writer io.Writer, sourceDir string) (tarErr error) {
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

	// Walk through the directory tree
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) (walkErr error) {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			return err
		}

		if !info.Mode().IsRegular() && !info.IsDir() {
			// Skip if it's not a regular file or not a directory
			return nil
		}

		// Set the name to be relative to sourceDir
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Create a header based on the file info
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header: %w", err)
		}

		// Convert Windows paths to tar format (using forward slashes)
		header.Name = filepath.ToSlash(relPath)
		header.Uid = 0
		header.Gid = 0
		header.Uname = ""
		header.Gname = ""
		// Write the header
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		if !info.Mode().IsRegular() {
			// Skip if it's not a regular file
			return nil
		}

		// Open the fp for reading
		fp, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", path, err)
		}
		defer func() {
			closeErr := fp.Close()
			if walkErr == nil {
				walkErr = closeErr
			}
		}()

		// Copy the file contents to the tar writer
		if _, err := io.Copy(tw, fp); err != nil {
			return fmt.Errorf("failed to write file content to tar: %w", err)
		}

		return nil
	})
}
