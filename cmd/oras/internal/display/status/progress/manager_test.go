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

package progress

import (
	"fmt"
	"testing"

	"oras.land/oras/cmd/oras/internal/display/status/console"
	"oras.land/oras/cmd/oras/internal/display/status/console/testutils"
)

func Test_manager_render(t *testing.T) {
	pty, device, err := testutils.NewPty()
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()
	m := &manager{
		console: &console.Console{Console: pty},
	}
	_, height := m.console.Size()
	for i := 0; i < height; i++ {
		if _, err := m.Add(); err != nil {
			t.Fatal(err)
		}
	}
	m.render()
	// validate
	var want []string
	for i := height; i > 0; i -= 2 {
		want = append(want, fmt.Sprintf("%dF%s", i, zeroStatus))
	}
	if err = testutils.MatchPty(pty, device, want...); err != nil {
		t.Fatal(err)
	}
}
