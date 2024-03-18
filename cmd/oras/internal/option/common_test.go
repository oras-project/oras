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

package option

import (
	"os"
	"reflect"
	"testing"

	"github.com/spf13/pflag"
)

func TestCommon_FlagsInit(t *testing.T) {
	var test struct {
		Common
	}

	ApplyFlags(&test, pflag.NewFlagSet("oras-test", pflag.ExitOnError))
}

func TestCommon_UpdateTTY(t *testing.T) {
	testTTY := &os.File{}
	tests := []struct {
		name        string
		flagPresent bool
		toSTDOUT    bool
		noTTY       bool
		expectedTTY *os.File
	}{
		{
			"output to STDOUT, --no-tty flag not used, reset TTY", false, true, false, nil,
		},
		{
			"output to STDOUT, --no-tty set to true, reset TTY", true, true, true, nil,
		},
		{
			"output to STDOUT, --no-tty set to false", true, true, false, testTTY,
		},
		{
			"not output to STDOUT, --no-tty flag not used", false, false, false, testTTY,
		},
		{
			"not output to STDOUT, --no-tty set to true, reset TTY", true, false, true, nil,
		},
		{
			"not output to STDOUT, --no-tty set to false", true, false, false, testTTY,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &Common{
				noTTY: tt.noTTY,
				TTY:   testTTY,
			}
			opts.UpdateTTY(tt.flagPresent, tt.toSTDOUT)
			if !reflect.DeepEqual(opts.TTY, tt.expectedTTY) {
				t.Fatalf("tt.TTY got %v, want %v", opts.TTY, tt.expectedTTY)
			}
		})
	}
}
