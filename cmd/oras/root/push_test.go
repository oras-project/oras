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
	"oras.land/oras-go/v2/content/oci"
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

// Test_runPush_force verifies that pushing with --force succeeds and wraps the
// destination in a contentutil.TraversingTarget (covering the force branch in
// runPush). Extra tags are also supplied so the wrapped originalDst is used by
// the extraRefs tagging path.
func Test_runPush_force(t *testing.T) {
	// prepare a file to push
	tempDir := t.TempDir()
	fileName := "hi.txt"
	filePath := filepath.Join(tempDir, fileName)
	if err := os.WriteFile(filePath, []byte("hello world"), 0600); err != nil {
		t.Fatal(err)
	}

	// use an OCI image layout as the push destination
	layoutDir := filepath.Join(tempDir, "layout")

	cmd := pushCmd()
	cmd.SetArgs([]string{
		"--oci-layout",
		"--force",
		"--disable-path-validation",
		layoutDir + ":tag1,tag2",
		filePath,
	})
	cmd.SetContext(context.Background())
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// verify the artifact and both tags were pushed to the layout
	store, err := oci.New(layoutDir)
	if err != nil {
		t.Fatalf("failed to open pushed layout: %v", err)
	}
	for _, tag := range []string{"tag1", "tag2"} {
		if _, err := store.Resolve(context.Background(), tag); err != nil {
			t.Errorf("expected tag %q to be resolvable, got error: %v", tag, err)
		}
	}
}
