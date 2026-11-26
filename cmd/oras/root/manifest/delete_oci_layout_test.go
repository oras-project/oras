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

package manifest

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oras-project/oras-go/v3"
	"github.com/oras-project/oras-go/v3/content/oci"
	"github.com/spf13/cobra"
)

func runManifestDelete(t *testing.T, args ...string) (string, error) {
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

func requireManifestLayoutNotExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("os.Stat(%q) error = %v, want fs.ErrNotExist", path, err)
	}
}

func Test_deleteCmd_missingOCILayout(t *testing.T) {
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
			const rawReference = "missing-layout:v1"
			args := append([]string{}, tt.flags...)
			args = append(args, "--oci-layout", rawReference)

			stdout, err := runManifestDelete(t, args...)
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
			requireManifestLayoutNotExist(t, "missing-layout")
		})
	}
}

func Test_deleteCmd_missingOCILayoutPath(t *testing.T) {
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
			const rawReference = "v1"
			args := append([]string{}, tt.flags...)
			args = append(args, "--oci-layout-path", layoutDir, rawReference)

			stdout, err := runManifestDelete(t, args...)
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
			if got := strings.Contains(stdout, "Missing "+rawReference); got != tt.wantMissing {
				t.Errorf("deleteCmd() output = %q, contains Missing = %v, want %v", stdout, got, tt.wantMissing)
			}
			requireManifestLayoutNotExist(t, layoutDir)
		})
	}
}

func Test_deleteCmd_missingOCILayout_invalidReference(t *testing.T) {
	t.Chdir(t.TempDir())

	stdout, err := runManifestDelete(t, "--force", "--oci-layout", "missing-layout:")
	if err == nil {
		t.Fatal("deleteCmd() error = nil, want reference validation error")
	}
	if !strings.Contains(err.Error(), "no tag or digest specified") {
		t.Errorf("deleteCmd() error = %v, want missing tag or digest error", err)
	}
	if strings.Contains(stdout, "Missing") {
		t.Errorf("deleteCmd() output = %q, want no Missing output", stdout)
	}
	requireManifestLayoutNotExist(t, "missing-layout")
}

func Test_deleteCmd_missingOCILayout_existingLayoutUnchanged(t *testing.T) {
	layoutDir := filepath.Join(t.TempDir(), "layout")
	ctx := context.Background()
	store, err := oci.New(layoutDir)
	if err != nil {
		t.Fatalf("oci.New() error = %v", err)
	}
	desc, err := oras.PackManifest(ctx, store, oras.PackManifestVersion1_1, "application/example", oras.PackManifestOptions{})
	if err != nil {
		t.Fatalf("oras.PackManifest() error = %v", err)
	}
	if err := store.Tag(ctx, desc, "v1"); err != nil {
		t.Fatalf("Store.Tag() error = %v", err)
	}
	beforeIndex := readManifestTestFile(t, filepath.Join(layoutDir, "index.json"))
	beforeLayout := readManifestTestFile(t, filepath.Join(layoutDir, "oci-layout"))

	stdout, err := runManifestDelete(t, "--force", "--oci-layout-path", layoutDir, "missing")
	if err != nil {
		t.Fatalf("deleteCmd() error = %v, want nil", err)
	}
	if !strings.Contains(stdout, "Missing missing") {
		t.Errorf("deleteCmd() output = %q, want Missing output", stdout)
	}
	if after := readManifestTestFile(t, filepath.Join(layoutDir, "index.json")); !bytes.Equal(beforeIndex, after) {
		t.Errorf("index.json changed: got %s, want %s", after, beforeIndex)
	}
	if after := readManifestTestFile(t, filepath.Join(layoutDir, "oci-layout")); !bytes.Equal(beforeLayout, after) {
		t.Errorf("oci-layout changed: got %s, want %s", after, beforeLayout)
	}
	reopened, err := oci.New(layoutDir)
	if err != nil {
		t.Fatalf("oci.New() error = %v", err)
	}
	if _, err := reopened.Resolve(ctx, "v1"); err != nil {
		t.Errorf("Resolve(%q) error = %v, want existing tagged manifest", "v1", err)
	}
}

func readManifestTestFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}
	return content
}
