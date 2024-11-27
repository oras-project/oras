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
	"os"
	"testing"

	containerd "github.com/containerd/console"
	"oras.land/oras/internal/testutils"
)

func givenConsole(t *testing.T) (c Console, pty containerd.Console) {
	pty, _, err := containerd.NewPty()
	if err != nil {
		t.Fatal(err)
	}

	c = &console{
		Console: pty,
	}
	return c, pty
}

func givenTestConsole(t *testing.T) (c Console, pty containerd.Console, tty *os.File) {
	var err error
	pty, tty, err = testutils.NewPty()
	if err != nil {
		t.Fatal(err)
	}
	c, err = NewConsole(tty)
	if err != nil {
		t.Fatal(err)
	}
	return
}

func validateSize(t *testing.T, gotWidth, gotHeight, wantWidth, wantHeight int) {
	t.Helper()
	if gotWidth != wantWidth {
		t.Errorf("Console.Size() gotWidth = %v, want %v", gotWidth, wantWidth)
	}
	if gotHeight != wantHeight {
		t.Errorf("Console.Size() gotHeight = %v, want %v", gotHeight, wantHeight)
	}
}

func TestNewConsole(t *testing.T) {
	_, err := NewConsole(os.Stdin)
	if err == nil {
		t.Error("expected error creating bogus console")
	}
}

func TestConsole_GetHeightWidth(t *testing.T) {
	c, pty := givenConsole(t)

	// minimal width and height
	gotHeight, gotWidth := c.GetHeightWidth()
	validateSize(t, gotWidth, gotHeight, MinWidth, MinHeight)

	// zero width
	_ = pty.Resize(containerd.WinSize{Width: 0, Height: MinHeight})
	gotHeight, gotWidth = c.GetHeightWidth()
	validateSize(t, gotWidth, gotHeight, MinWidth, MinHeight)

	// zero height
	_ = pty.Resize(containerd.WinSize{Width: MinWidth, Height: 0})
	gotHeight, gotWidth = c.GetHeightWidth()
	validateSize(t, gotWidth, gotHeight, MinWidth, MinHeight)

	// valid zero and height
	_ = pty.Resize(containerd.WinSize{Width: 200, Height: 100})
	gotHeight, gotWidth = c.GetHeightWidth()
	validateSize(t, gotWidth, gotHeight, 200, 100)

}

func TestConsole_NewRow(t *testing.T) {
	c, pty, tty := givenTestConsole(t)

	c.NewRow()

	err := testutils.MatchPty(pty, tty, "\x1b8\r\n\x1b7")
	if err != nil {
		t.Fatalf("NewRow output error: %v", err)
	}
}

func TestConsole_OutputTo(t *testing.T) {
	c, pty, tty := givenTestConsole(t)

	c.OutputTo(1, "test string")

	err := testutils.MatchPty(pty, tty, "\x1b8\x1b[1Ftest string\x1b[0m\r\n\x1b[0K")
	if err != nil {
		t.Fatalf("OutputTo output error: %v", err)
	}
}

func TestConsole_Restore(t *testing.T) {
	c, pty, tty := givenTestConsole(t)

	c.Restore()

	err := testutils.MatchPty(pty, tty, "\x1b8\x1b[0G\x1b[2K\x1b[?25h")
	if err != nil {
		t.Fatalf("Restore output error: %v", err)
	}
}

func TestConsole_Save(t *testing.T) {
	c, pty, tty := givenTestConsole(t)

	c.Save()

	err := testutils.MatchPty(pty, tty, "\x1b[?25l\x1b7\x1b[0m")
	if err != nil {
		t.Fatalf("Save output error: %v", err)
	}
}
