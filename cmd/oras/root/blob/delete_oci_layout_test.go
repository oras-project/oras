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

package blob

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/oras-project/oras-go/v3/content/oci"
	"github.com/spf13/cobra"
)

func runBlobDelete(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "oras"}
	root.AddCommand(deleteCmd())
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(append([]string{"delete"}, args...))
	err := root.Execute()
	return stdout.String(), err
}

func requireBlobLayoutNotExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("os.Stat(%q) error = %v, want fs.ErrNotExist", path, err)
	}
}

func Test_deleteCmd_missingOCILayout(t *testing.T) {
	missingDigest := "sha256:" + strings.Repeat("0", 64)
	tests := []struct {
		name        string
		flags       []string
		wantMissing bool
		wantErr     error
	}{
		{
			name:        "force",
			flags:       []string{"--force"},
			wantMissing: true,
		},
		{
			name:    "without force",
			wantErr: fs.ErrNotExist,
		},
		{
			name:    "force with descriptor",
			flags:   []string{"--force", "--descriptor"},
			wantErr: fs.ErrNotExist,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			rawReference := "missing-layout@" + missingDigest
			args := append([]string{}, tt.flags...)
			args = append(args, "--oci-layout", rawReference)

			stdout, err := runBlobDelete(t, args...)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("deleteCmd() error = %v, want error wrapping %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("deleteCmd() error = %v, want nil", err)
			}
			if got := strings.Contains(stdout, "Missing "+rawReference); got != tt.wantMissing {
				t.Errorf("deleteCmd() output = %q, contains Missing = %v, want %v", stdout, got, tt.wantMissing)
			}
			requireBlobLayoutNotExist(t, "missing-layout")
		})
	}
}

func Test_deleteCmd_missingOCILayoutPath(t *testing.T) {
	missingDigest := "sha256:" + strings.Repeat("0", 64)
	tests := []struct {
		name        string
		flags       []string
		wantMissing bool
		wantErr     error
	}{
		{
			name:        "force",
			flags:       []string{"--force"},
			wantMissing: true,
		},
		{
			name:    "without force",
			wantErr: fs.ErrNotExist,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layoutDir := filepath.Join(t.TempDir(), "missing-layout")
			args := append([]string{}, tt.flags...)
			args = append(args, "--oci-layout-path", layoutDir, missingDigest)

			stdout, err := runBlobDelete(t, args...)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("deleteCmd() error = %v, want error wrapping %v", err, tt.wantErr)
				}
				if strings.Contains(strings.ToLower(err.Error()), "registry:") {
					t.Errorf("deleteCmd() error = %v, want no registry error prefix", err)
				}
			} else if err != nil {
				t.Fatalf("deleteCmd() error = %v, want nil", err)
			}
			if got := strings.Contains(stdout, "Missing "+missingDigest); got != tt.wantMissing {
				t.Errorf("deleteCmd() output = %q, contains Missing = %v, want %v", stdout, got, tt.wantMissing)
			}
			requireBlobLayoutNotExist(t, layoutDir)
		})
	}
}

func Test_deleteCmd_missingOCILayout_invalidReference(t *testing.T) {
	t.Chdir(t.TempDir())

	stdout, err := runBlobDelete(t, "--force", "--oci-layout", "missing-layout@")
	if err == nil {
		t.Fatal("deleteCmd() error = nil, want reference validation error")
	}
	if !strings.Contains(err.Error(), "no digest specified") {
		t.Errorf("deleteCmd() error = %v, want missing digest error", err)
	}
	if strings.Contains(stdout, "Missing") {
		t.Errorf("deleteCmd() output = %q, want no Missing output", stdout)
	}
	requireBlobLayoutNotExist(t, "missing-layout")
}

func Test_deleteCmd_missingOCILayout_existingLayoutUnchanged(t *testing.T) {
	layoutDir := filepath.Join(t.TempDir(), "layout")
	ctx := context.Background()
	store, err := oci.New(layoutDir)
	if err != nil {
		t.Fatalf("oci.New() error = %v", err)
	}
	content := []byte("blob content")
	desc := ocispec.Descriptor{
		MediaType: "application/vnd.test.blob",
		Digest:    digest.FromBytes(content),
		Size:      int64(len(content)),
	}
	if err := store.Push(ctx, desc, bytes.NewReader(content)); err != nil {
		t.Fatalf("Store.Push() error = %v", err)
	}
	beforeIndex := readBlobTestFile(t, filepath.Join(layoutDir, "index.json"))
	beforeLayout := readBlobTestFile(t, filepath.Join(layoutDir, "oci-layout"))

	stdout, err := runBlobDelete(t, "--force", "--oci-layout-path", layoutDir, missingBlobDigest)
	if err != nil {
		t.Fatalf("deleteCmd() error = %v, want nil", err)
	}
	if !strings.Contains(stdout, "Missing "+missingBlobDigest) {
		t.Errorf("deleteCmd() output = %q, want Missing output", stdout)
	}
	if after := readBlobTestFile(t, filepath.Join(layoutDir, "index.json")); !bytes.Equal(beforeIndex, after) {
		t.Errorf("index.json changed: got %s, want %s", after, beforeIndex)
	}
	if after := readBlobTestFile(t, filepath.Join(layoutDir, "oci-layout")); !bytes.Equal(beforeLayout, after) {
		t.Errorf("oci-layout changed: got %s, want %s", after, beforeLayout)
	}
	blobPath := filepath.Join(layoutDir, "blobs", desc.Digest.Algorithm().String(), desc.Digest.Encoded())
	if _, err := os.Stat(blobPath); err != nil {
		t.Errorf("os.Stat(%q) error = %v, want existing blob", blobPath, err)
	}
}

func readBlobTestFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}
	return content
}

func Test_deleteCmd_missingRemoteCAFile(t *testing.T) {
	t.Chdir(t.TempDir())
	missingCA := filepath.Join(t.TempDir(), "missing-ca.pem")
	missingDigest := "sha256:" + strings.Repeat("0", 64)

	stdout, err := runBlobDelete(t, "--force", "--ca-file", missingCA, "localhost:5000/test@"+missingDigest)
	if err == nil {
		t.Fatal("deleteCmd() error = nil, want missing CA file error")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("deleteCmd() error = %v, want error wrapping fs.ErrNotExist", err)
	}
	if strings.Contains(stdout, "Missing") {
		t.Errorf("deleteCmd() output = %q, want no Missing output", stdout)
	}
}

func Test_deleteCmd_existingInvalidOCILayout(t *testing.T) {
	missingDigest := "sha256:" + strings.Repeat("0", 64)
	layoutContent := []byte(`{"imageLayoutVersion":"1.0.0"}`)
	tests := []struct {
		name       string
		prepare    func(t *testing.T, dir string)
		wantKeep   bool
		wantLayout bool
	}{
		{
			name:    "empty directory",
			prepare: func(*testing.T, string) {},
		},
		{
			name: "non-layout directory",
			prepare: func(t *testing.T, dir string) {
				if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("keep"), 0600); err != nil {
					t.Fatalf("os.WriteFile() error = %v", err)
				}
			},
			wantKeep: true,
		},
		{
			name: "missing index.json",
			prepare: func(t *testing.T, dir string) {
				if err := os.WriteFile(filepath.Join(dir, "oci-layout"), layoutContent, 0600); err != nil {
					t.Fatalf("os.WriteFile() error = %v", err)
				}
			},
			wantLayout: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layoutDir := filepath.Join(t.TempDir(), "layout")
			if err := os.Mkdir(layoutDir, 0700); err != nil {
				t.Fatalf("os.Mkdir() error = %v", err)
			}
			tt.prepare(t, layoutDir)

			stdout, err := runBlobDelete(t, "--force", "--oci-layout-path", layoutDir, missingDigest)
			if err == nil {
				t.Fatal("deleteCmd() error = nil, want invalid OCI layout error")
			}
			if !strings.Contains(err.Error(), "not an OCI image layout") {
				t.Errorf("deleteCmd() error = %v, want invalid OCI layout error", err)
			}
			if strings.Contains(stdout, "Missing") {
				t.Errorf("deleteCmd() output = %q, want no Missing output", stdout)
			}
			if tt.wantLayout {
				if got := readBlobTestFile(t, filepath.Join(layoutDir, "oci-layout")); !bytes.Equal(got, layoutContent) {
					t.Errorf("oci-layout changed: got %q, want %q", got, layoutContent)
				}
				for _, name := range []string{"index.json", "blobs"} {
					if _, err := os.Stat(filepath.Join(layoutDir, name)); !errors.Is(err, fs.ErrNotExist) {
						t.Errorf("os.Stat(%q) error = %v, want fs.ErrNotExist", name, err)
					}
				}
			} else {
				for _, name := range []string{"oci-layout", "index.json", "blobs"}{
					if _, err := os.Stat(filepath.Join(layoutDir, name)); !errors.Is(err, fs.ErrNotExist) {
						t.Errorf("os.Stat(%q) error = %v, want fs.ErrNotExist", name, err)
					}
				}
			}
			_, keepErr := os.Stat(filepath.Join(layoutDir, "keep.txt"))
			if tt.wantKeep && keepErr != nil {
				t.Errorf("os.Stat(%q) error = %v, want existing file", "keep.txt", keepErr)
			}
			if !tt.wantKeep && !errors.Is(keepErr, fs.ErrNotExist) {
				t.Errorf("os.Stat(%q) error = %v, want fs.ErrNotExist", "keep.txt", keepErr)
			}
		})
	}
}

func Test_deleteCmd_existingOCILayout_deletesBlob(t *testing.T) {
	layoutDir := filepath.Join(t.TempDir(), "layout")
	ctx := context.Background()
	store, err := oci.New(layoutDir)
	if err != nil {
		t.Fatalf("oci.New() error = %v", err)
	}
	content := []byte("blob content")
	desc := ocispec.Descriptor{
		MediaType: "application/vnd.test.blob",
		Digest:    digest.FromBytes(content),
		Size:      int64(len(content)),
	}
	if err := store.Push(ctx, desc, bytes.NewReader(content)); err != nil {
		t.Fatalf("Store.Push() error = %v", err)
	}

	stdout, err := runBlobDelete(t, "--force", "--oci-layout-path", layoutDir, desc.Digest.String())
	if err != nil {
		t.Fatalf("deleteCmd() error = %v, want nil", err)
	}
	if !strings.Contains(stdout, "Deleted") {
		t.Errorf("deleteCmd() output = %q, want Deleted output", stdout)
	}
	blobPath := filepath.Join(layoutDir, "blobs", desc.Digest.Algorithm().String(), desc.Digest.Encoded())
	if _, err := os.Stat(blobPath); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("os.Stat(%q) error = %v, want fs.ErrNotExist", blobPath, err)
	}
}
