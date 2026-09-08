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

package credential

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestNewStoreMalformedConfig covers the error path on every platform: JSON
// decoding fails the same way everywhere, unlike file permissions.
func TestNewStoreMalformedConfig(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(filename, []byte("{not json"), 0600); err != nil {
		t.Fatalf("cannot create config file: %v", err)
	}

	credStore, err := NewStore(filename)
	if credStore != nil {
		t.Errorf("expected NewStore to return a nil store, got %v", credStore)
	}
	if err == nil {
		t.Fatal("expected NewStore to return an error, got nil")
	}
	if want := "failed to decode config file"; !strings.Contains(err.Error(), want) {
		t.Errorf("expected error to contain %q, got %q", want, err.Error())
	}
}

func TestNewStoreError(t *testing.T) {
	if runtime.GOOS == "windows" {
		// os.Chmod on Windows only toggles the read-only attribute and cannot
		// clear the read permission, so the file stays readable and NewStore
		// succeeds. See https://pkg.go.dev/os#Chmod.
		t.Skip("os.Chmod cannot make a file unreadable on Windows")
	}
	if os.Geteuid() == 0 {
		// Permission bits do not restrict root, so the file stays readable.
		t.Skip("root bypasses file permissions")
	}

	filename := filepath.Join(t.TempDir(), "testfile.txt")
	file, err := os.Create(filename)
	if err != nil {
		t.Fatalf("cannot create file: %v", err)
	}
	defer func() { _ = file.Close() }()

	if err := os.Chmod(filename, 000); err != nil {
		t.Fatalf("cannot change file permissions: %v", err)
	}

	credStore, err := NewStore(filename)
	if credStore != nil {
		t.Errorf("expected NewStore to return a nil store, got %v", credStore)
	}
	if err == nil {
		t.Fatal("expected NewStore to return an error, got nil")
	}
	if want := "failed to open config file"; !strings.Contains(err.Error(), want) {
		t.Errorf("expected error to contain %q, got %q", want, err.Error())
	}
}
