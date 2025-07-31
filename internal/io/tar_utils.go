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
	"fmt"
	"io"
	"os"
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
