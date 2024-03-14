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
	"testing"

	"oras.land/oras/cmd/oras/internal/display/status/console/testutils"
)

func TestCommon_parseTTY(t *testing.T) {
	_, device, err := testutils.NewPty()
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()
	var opts Common

	// TTY output
	if err := opts.parseTTY(device); err != nil {
		t.Errorf("unexpected error with TTY output: %v", err)
	}

	// --debug
	opts.Debug = true
	if err := opts.parseTTY(device); err != nil {
		t.Errorf("unexpected error with --debug: %v", err)
	}
	if !opts.noTTY {
		t.Errorf("expected --no-tty to be true with --debug")
	}
}
