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
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	oraserrors "oras.land/oras/cmd/oras/internal/errors"
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
	want := oraserrors.UnsupportedFormatTypeError(opts.Format.Type).Error()
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
	want := oraserrors.UnsupportedFormatTypeError(opts.Format.Type).Error()
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

func Test_runPushRecursive_walkError(t *testing.T) {
	// prepare
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	// test with non-existent directory
	opts := &pushOptions{
		Recursive: option.Recursive{
			Recursive:           true,
			MaxBlobsPerManifest: 1000,
		},
	}
	opts.FileRefs = []string{"/nonexistent/path/that/does/not/exist"}

	err := runPushRecursive(cmd, opts)
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
}

func Test_pushArtifact_packError(t *testing.T) {
	packErr := errors.New("pack error")
	pack := func() (ocispec.Descriptor, error) {
		return ocispec.Descriptor{}, packErr
	}
	copyFn := func(desc ocispec.Descriptor) error {
		return nil
	}

	_, err := pushArtifact(nil, pack, copyFn)
	if err == nil {
		t.Fatal("expected error from pack")
	}
	if !errors.Is(err, packErr) {
		t.Fatalf("got error %v, want %v", err, packErr)
	}
}

func Test_pushArtifact_copyError(t *testing.T) {
	copyErr := errors.New("copy error")
	pack := func() (ocispec.Descriptor, error) {
		return ocispec.Descriptor{
			MediaType: "application/vnd.oci.image.manifest.v1+json",
			Digest:    "sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			Size:      100,
		}, nil
	}
	copyFn := func(desc ocispec.Descriptor) error {
		return copyErr
	}

	_, err := pushArtifact(nil, pack, copyFn)
	if err == nil {
		t.Fatal("expected error from copy")
	}
	if !errors.Is(err, copyErr) {
		t.Fatalf("got error %v, want %v", err, copyErr)
	}
}

func Test_pushArtifact_success(t *testing.T) {
	expectedDesc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    "sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		Size:      100,
	}
	pack := func() (ocispec.Descriptor, error) {
		return expectedDesc, nil
	}
	copyFn := func(desc ocispec.Descriptor) error {
		return nil
	}

	desc, err := pushArtifact(nil, pack, copyFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if desc.Digest != expectedDesc.Digest {
		t.Fatalf("got digest %v, want %v", desc.Digest, expectedDesc.Digest)
	}
}

func Test_pushCmd(t *testing.T) {
	cmd := pushCmd()

	// Check command basics
	if cmd.Use == "" {
		t.Error("command Use is empty")
	}
	if cmd.Short == "" {
		t.Error("command Short is empty")
	}
	if cmd.Long == "" {
		t.Error("command Long is empty")
	}

	// Check flags are registered
	flags := []string{
		"config",
		"artifact-type",
		"concurrency",
		"recursive",
		"max-blobs-per-manifest",
		"preserve-empty-dirs",
		"follow-symlinks",
	}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("flag %q not registered", flag)
		}
	}
}

func Test_pushCmd_PreRunE_recursiveValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		setup   func(t *testing.T) string
		flags   map[string]string
		wantErr string
	}{
		{
			name: "recursive with config error",
			args: []string{"localhost:5000/test:v1", "somedir"},
			setup: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			flags: map[string]string{
				"recursive": "true",
				"config":    "config.json",
			},
			wantErr: "--config cannot be used with --recursive",
		},
		{
			name: "recursive with multiple args error",
			args: []string{"localhost:5000/test:v1", "dir1", "dir2"},
			setup: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			flags: map[string]string{
				"recursive": "true",
			},
			wantErr: "--recursive requires exactly one directory argument",
		},
		{
			name: "recursive with file instead of dir",
			args: []string{"localhost:5000/test:v1"},
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				file := filepath.Join(dir, "file.txt")
				_ = os.WriteFile(file, []byte("test"), 0644)
				return file
			},
			flags: map[string]string{
				"recursive":               "true",
				"disable-path-validation": "true",
			},
			wantErr: "is not a directory; --recursive requires a directory",
		},
		{
			name: "recursive with nonexistent path",
			args: []string{"localhost:5000/test:v1", "nonexistent_path"},
			setup: func(t *testing.T) string {
				t.Helper()
				return ""
			},
			flags: map[string]string{
				"recursive": "true",
			},
			wantErr: "cannot access",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := pushCmd()
			cmd.SetContext(context.Background())

			path := tt.setup(t)

			// Build args
			args := tt.args
			if path != "" && len(args) == 1 {
				args = append(args, path)
			}

			// Set flags
			for k, v := range tt.flags {
				if err := cmd.Flags().Set(k, v); err != nil {
					t.Fatalf("failed to set flag %s: %v", k, err)
				}
			}

			// Run PreRunE
			err := cmd.PreRunE(cmd, args)
			if err == nil {
				t.Fatal("expected error")
			}
			if tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}
