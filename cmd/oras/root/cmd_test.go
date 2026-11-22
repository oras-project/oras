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
	"strconv"
	"testing"

	"github.com/spf13/cobra"
)

func newConcurrencyTestCmd(level int) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Int("concurrency", level, "concurrency level")
	return cmd
}

func Test_validateConcurrency(t *testing.T) {
	tests := []struct {
		name    string
		level   int
		set     bool
		wantErr bool
	}{
		{name: "flag not used", level: 5},
		{name: "positive level", level: 9, set: true},
		{name: "zero means no limit", level: 0, set: true},
		{name: "negative level", level: -3, set: true, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newConcurrencyTestCmd(tt.level)
			if tt.set {
				if err := cmd.Flags().Set("concurrency", strconv.Itoa(tt.level)); err != nil {
					t.Fatalf("unexpected error setting flag: %v", err)
				}
			}

			err := validateConcurrency(cmd)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				want := "`--concurrency -3` is invalid: concurrency level must not be negative (use 0 for no limit)"
				if err.Error() != want {
					t.Fatalf("expected error %q, got %q", want, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
