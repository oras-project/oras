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
	"testing"

	"github.com/spf13/cobra"
)

func Test_concurrencyFlag_validationAndDefaults(t *testing.T) {
	tests := []struct {
		name        string
		newCmd      func() *cobra.Command
		wantDefault string
		args        []string
	}{
		{
			name:        "attach",
			newCmd:      attachCmd,
			wantDefault: "5",
			args:        []string{"--artifact-type", "doc/example", "localhost:5000/hello:v1", "hi.txt"},
		},
		{
			name:        "backup",
			newCmd:      backupCmd,
			wantDefault: "3",
			args:        []string{"--output", "out", "localhost:5000/hello:v1"},
		},
		{
			name:        "cp",
			newCmd:      copyCmd,
			wantDefault: "3",
			args:        []string{"localhost:5000/hello:v1", "localhost:5000/hello:v2"},
		},
		{
			name:        "pull",
			newCmd:      pullCmd,
			wantDefault: "3",
			args:        []string{"localhost:5000/hello:v1"},
		},
		{
			name:        "push",
			newCmd:      pushCmd,
			wantDefault: "5",
			args:        []string{"localhost:5000/hello:v1", "hi.txt"},
		},
		{
			name:        "restore",
			newCmd:      restoreCmd,
			wantDefault: "5",
			args:        []string{"--input", "in", "localhost:5000"},
		},
		{
			name:        "tag",
			newCmd:      tagCmd,
			wantDefault: "5",
			args:        []string{"localhost:5000/hello:v1", "v2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.newCmd().Flags().Lookup("concurrency")
			if f == nil {
				t.Fatalf("%s: expected --concurrency flag to be registered", tt.name)
			}
			if f.DefValue != tt.wantDefault {
				t.Fatalf("%s: --concurrency default = %q, want %q", tt.name, f.DefValue, tt.wantDefault)
			}
			for _, val := range []string{"0", "-1"} {
				cmd := tt.newCmd()
				cmd.SetArgs(append([]string{"--concurrency", val}, tt.args...))
				err := cmd.Execute()
				if err == nil || err.Error() != "concurrency value should be at least 1" {
					t.Fatalf("%s --concurrency %s: got %v, want concurrency validation error", tt.name, val, err)
				}
			}
		})
	}
}
