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
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestConcurrencyFlagRejectsNegativeLevels checks every command that passes
// --concurrency to the copy or tag paths, and that check keeps accepting a
// non-positive level because its flag help documents that as the default.
func TestConcurrencyFlagRejectsNegativeLevels(t *testing.T) {
	dst := "localhost:5000/hello:v1"
	tests := []struct {
		name string
		cmd  func() *cobra.Command
		args []string
	}{
		{"attach", attachCmd, []string{dst, "hi.txt"}},
		{"backup", backupCmd, []string{"backup.tar"}},
		{"cp", copyCmd, []string{dst, "localhost:5000/hello:v2"}},
		{"pull", pullCmd, []string{dst}},
		{"push", pushCmd, []string{dst, "hi.txt"}},
		{"restore", restoreCmd, []string{"backup.tar"}},
		{"tag", tagCmd, []string{dst, "v2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.cmd()
			if err := cmd.Flags().Parse([]string{"--concurrency", "-3"}); err != nil {
				t.Fatalf("parsing flags: %v", err)
			}
			err := cmd.PreRunE(cmd, tt.args)
			if err == nil {
				t.Fatal("expected an error for a negative concurrency level")
			}
			want := "`--concurrency -3` is invalid: concurrency level must not be negative"
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("got %q, want it to contain %q", err, want)
			}
		})
	}
}

// TestCheckConcurrencyAllowsNonPositive documents that check keeps its
// documented behaviour ("0 or less uses the default").
func TestCheckConcurrencyAllowsNonPositive(t *testing.T) {
	cmd := checkCmd()
	if err := cmd.Flags().Parse([]string{"--concurrency", "-1"}); err != nil {
		t.Fatalf("parsing flags: %v", err)
	}
	err := cmd.PreRunE(cmd, []string{"localhost:5000/hello:v1"})
	if err != nil && strings.Contains(err.Error(), "--concurrency") {
		t.Fatalf("check must keep accepting a non-positive level, got %v", err)
	}
}
