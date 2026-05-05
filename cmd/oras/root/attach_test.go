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
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"oras.land/oras/cmd/oras/internal/option"
)

func Test_attachCmd_configAndPlatformMutuallyExclusive(t *testing.T) {
	cmd := attachCmd()
	cmd.SetArgs([]string{
		"--artifact-type", "doc/example",
		"--config", "config.json",
		"--platform", "linux/amd64",
		"localhost:5000/hello:v1",
		"hi.txt",
	})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "cannot be used at the same time") {
		t.Fatalf("expected mutual exclusion error, got %v", err)
	}
}

func Test_attachCmd_configFlagRegistered(t *testing.T) {
	cmd := attachCmd()
	f := cmd.Flags().Lookup("config")
	if f == nil {
		t.Fatal("expected --config flag to be registered")
	}
	if f.Usage != "`path` of image config file" {
		t.Fatalf("unexpected usage: %q", f.Usage)
	}
}

func Test_runAttach_errType(t *testing.T) {
	// prepare
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	// test
	opts := &attachOptions{
		Packer: option.Packer{
			Annotation: option.Annotation{
				ManifestAnnotations: []string{"one", "two"},
			},
			AnnotationFilePath: "/tmp/whatever",
		},
	}
	got := opts.Packer.Parse(cmd).Error()
	want := errors.New("`--annotation` and `--annotation-file` cannot be both specified").Error()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}
