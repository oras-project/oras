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

package console

import (
	"testing"

	"github.com/containerd/console"
)

func validateSize(t *testing.T, gotWidth, gotHeight, wantWidth, wantHeight int) {
	t.Helper()
	if gotWidth != wantWidth {
		t.Errorf("Console.Size() gotWidth = %v, want %v", gotWidth, wantWidth)
	}
	if gotHeight != wantHeight {
		t.Errorf("Console.Size() gotHeight = %v, want %v", gotHeight, wantHeight)
	}
}

func TestConsole_Size(t *testing.T) {
	pty, _, err := console.NewPty()
	if err != nil {
		t.Fatal(err)
	}
	c := &Console{
		Console: pty,
	}

	// minimal width and height
	gotWidth, gotHeight := c.Size()
	validateSize(t, gotWidth, gotHeight, MinWidth, MinHeight)

	// zero width
	_ = pty.Resize(console.WinSize{Width: 0, Height: MinHeight})
	gotWidth, gotHeight = c.Size()
	validateSize(t, gotWidth, gotHeight, MinWidth, MinHeight)

	// zero height
	_ = pty.Resize(console.WinSize{Width: MinWidth, Height: 0})
	gotWidth, gotHeight = c.Size()
	validateSize(t, gotWidth, gotHeight, MinWidth, MinHeight)

	// valid zero and height
	_ = pty.Resize(console.WinSize{Width: 200, Height: 100})
	gotWidth, gotHeight = c.Size()
	validateSize(t, gotWidth, gotHeight, 200, 100)
}
