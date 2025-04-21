//go:build freebsd || linux || netbsd || openbsd || solaris

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

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"oras.land/oras/internal/testutils"
)

func TestNotty_FlagsInit(t *testing.T) {
	var test struct {
		Common
	}

	ApplyFlags(&test, pflag.NewFlagSet("oras-test", pflag.ExitOnError))
}

func TestCommon_ParseTTY(t *testing.T) {
	_, device, err := testutils.NewPty()
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()
	opts := NoTTY{
		noTTY: false,
		TTY:   device,
	}
	// TTY output
	if err := opts.Parse(&cobra.Command{}); err != nil {
		t.Errorf("unexpected error with TTY output: %v", err)
	}
}

func TestNotty_UpdateTTY(t *testing.T) {
	testTTY := &os.File{}
	tests := []struct {
		name        string
		debug       bool
		flagPresent bool
		toSTDOUT    bool
		noTTY       bool
		expectedTTY *os.File
	}{
		{
			"output to STDOUT, --no-tty flag not used, reset TTY", false, false, true, false, nil,
		},
		{
			"output to STDOUT, --no-tty set to true, reset TTY", false, true, true, true, nil,
		},
		{
			"output to STDOUT, --no-tty set to false", false, true, true, false, testTTY,
		},
		{
			"not output to STDOUT, --no-tty flag not used", false, false, false, false, testTTY,
		},
		{
			"not output to STDOUT, --no-tty set to true, reset TTY", false, true, false, true, nil,
		},
		{
			"not output to STDOUT, --no-tty set to false", false, true, false, false, testTTY,
		},
		{
			"debug enabled, --no-tty flag is not used", true, false, false, false, nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &NoTTY{
				noTTY: tt.noTTY,
				TTY:   testTTY,
			}
			opts.UpdateTTY(tt.debug, tt.flagPresent, tt.toSTDOUT)
			if !reflect.DeepEqual(opts.TTY, tt.expectedTTY) {
				t.Fatalf("tt.TTY got %v, want %v", opts.TTY, tt.expectedTTY)
			}
		})
	}
}
