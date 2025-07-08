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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
)

func Test_runPull_errType(t *testing.T) {
	// prpare
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	// test
	opts := &pullOptions{
		Format: option.Format{
			Type: "unknown",
		},
	}
	got := runPull(cmd, opts).Error()
	want := errors.UnsupportedFormatTypeError(opts.Format.Type).Error()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func Test_doPull(t *testing.T) {
	t.Run("PreservePermissions", func(t *testing.T) {
		repo := "pp"
		tagReference := "dppp"
		// Create a temporary directory, put a file in there with mode 0777
		srcRoot, err := os.MkdirTemp("", "doPull_preservePermissions_srcDir-*")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.RemoveAll(srcRoot) }()

		srcDir, err := os.MkdirTemp(srcRoot, "*")
		if err != nil {
			t.Fatal(err)
		}
		tf, err := os.OpenFile(fmt.Sprintf("%s/foo.sh", srcDir), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0777)
		if err != nil {
			t.Fatal(err)
		}
		oldFs, err := tf.Stat()
		if err != nil {
			t.Fatal(err)
		}

		oldWd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chdir(oldWd) }()

		//fmt.Println(filepath.Base(srcDir))
		subDir := filepath.Base(srcDir)

		pushArgs := []string{
			"push",
			"--insecure",
			"--plain-http",
			"--no-tty",
			"--format",
			"go-template=\n",
			fmt.Sprintf("%s/%s:%s", genericHost, repo, tagReference),
			subDir,
		}
		cmd := New()
		cmd.SetArgs(pushArgs)
		os.Chdir(srcRoot)
		_, err = cmd.ExecuteC()
		if err != nil {
			t.Fatal(err)
		}

		tgtDir, err := os.MkdirTemp("", "doPull_preservePermissions_tgtDir-*")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.RemoveAll(tgtDir) }()

		pullArgs := []string{
			"pull",
			"--insecure",
			"--plain-http",
			"--no-tty",
			"--format",
			"go-template=\n",
			fmt.Sprintf("%s/%s:%s", genericHost, repo, tagReference),
			"-o",
			tgtDir,
		}
		cmd.SetArgs(pullArgs)
		_, err = cmd.ExecuteC()
		if err != nil {
			t.Fatal(err)
		}

		if fileStat, err := os.Stat(filepath.Join(tgtDir, subDir, "foo.sh")); err == nil {
			if oldFs.Mode().Perm() == fileStat.Mode().Perm() {
				return
			}
		}
		t.Fatal("failed to create file with correct permissions.")
	})
}
