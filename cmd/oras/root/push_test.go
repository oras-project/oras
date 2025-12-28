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

package root

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
)

func Test_runPush_errType(t *testing.T) {
	// prepare
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	// test
	opts := &pushOptions{
		Format: option.Format{
			Type: "unknown",
		},
	}
	got := runPush(cmd, opts).Error()
	want := errors.UnsupportedFormatTypeError(opts.Format.Type).Error()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func Test_runPushRecursive_errType(t *testing.T) {
	// prepare
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// test
	opts := &pushOptions{
		Format: option.Format{
			Type: "unknown",
		},
		Recursive: option.Recursive{
			Recursive:           true,
			MaxBlobsPerManifest: 1000,
		},
	}
	opts.FileRefs = []string{tmpDir}

	got := runPushRecursive(cmd, opts).Error()
	want := errors.UnsupportedFormatTypeError(opts.Format.Type).Error()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func Test_runPushRecursive_emptyDir(t *testing.T) {
	// prepare
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	tmpDir := t.TempDir()

	// test with empty directory and no preserve-empty-dirs
	opts := &pushOptions{
		Recursive: option.Recursive{
			Recursive:           true,
			MaxBlobsPerManifest: 1000,
			PreserveEmptyDirs:   false,
		},
	}
	opts.FileRefs = []string{tmpDir}

	err := runPushRecursive(cmd, opts)
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
	if err.Error() != "directory is empty; nothing to push" {
		t.Fatalf("got error %q, want %q", err.Error(), "directory is empty; nothing to push")
	}
}
