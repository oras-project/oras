//go:build !windows

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

	"github.com/creack/pty"
	"oras.land/oras/internal/testutils"
)

func givenConsole(t *testing.T) (c Console, ptmx *os.File) {
	t.Helper()
	ptmx, pts, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pts.Close() }()

	c = &console{
		file: ptmx,
	}
	return c, ptmx
}

func givenTestConsole(t *testing.T) (c Console, ptmx *os.File, pts *os.File) {
	t.Helper()
	var err error
	ptmx, pts, err = testutils.NewPty()
	if err != nil {
		t.Fatal(err)
	}
	c, err = NewConsole(pts)
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
	c, ptmx := givenConsole(t)
	defer func() { _ = ptmx.Close() }()

	// minimal width and height
	gotHeight, gotWidth := c.GetHeightWidth()
	validateSize(t, gotWidth, gotHeight, MinWidth, MinHeight)

	// zero width
	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: MinHeight, Cols: 0})
	gotHeight, gotWidth = c.GetHeightWidth()
	validateSize(t, gotWidth, gotHeight, MinWidth, MinHeight)

	// zero height
	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 0, Cols: MinWidth})
	gotHeight, gotWidth = c.GetHeightWidth()
	validateSize(t, gotWidth, gotHeight, MinWidth, MinHeight)

	// valid width and height
	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 100, Cols: 200})
	gotHeight, gotWidth = c.GetHeightWidth()
	validateSize(t, gotWidth, gotHeight, 200, 100)

}

func TestConsole_NewRow(t *testing.T) {
	c, ptmx, pts := givenTestConsole(t)
	defer func() { _ = ptmx.Close() }()

	c.NewRow()

	err := testutils.MatchPty(ptmx, pts, "\x1b8\r\n\x1b7")
	if err != nil {
		t.Fatalf("NewRow output error: %v", err)
	}
}

func TestConsole_OutputTo(t *testing.T) {
	c, ptmx, pts := givenTestConsole(t)
	defer func() { _ = ptmx.Close() }()

	c.OutputTo(1, "test string")

	err := testutils.MatchPty(ptmx, pts, "\x1b8\x1b[1Ftest string\x1b[0m\r\n\x1b[0K")
	if err != nil {
		t.Fatalf("OutputTo output error: %v", err)
	}
}

func TestConsole_Restore(t *testing.T) {
	c, ptmx, pts := givenTestConsole(t)
	defer func() { _ = ptmx.Close() }()

	c.Restore()

	err := testutils.MatchPty(ptmx, pts, "\x1b8\x1b[0G\x1b[2K\x1b[?25h")
	if err != nil {
		t.Fatalf("Restore output error: %v", err)
	}
}

func TestConsole_Save(t *testing.T) {
	c, ptmx, pts := givenTestConsole(t)
	defer func() { _ = ptmx.Close() }()

	c.Save()

	err := testutils.MatchPty(ptmx, pts, "\x1b[?25l\x1b7\x1b[0m")
	if err != nil {
		t.Fatalf("Save output error: %v", err)
	}
}

func TestConsole_Write(t *testing.T) {
	ptmx, pts, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = ptmx.Close()
		_ = pts.Close()
	}()

	c := &console{file: ptmx}
	testData := []byte("test data")
	n, err := c.write(testData)
	if err != nil {
		t.Fatalf("write() error = %v, want nil", err)
	}
	if n != len(testData) {
		t.Errorf("write() returned %d bytes, want %d", n, len(testData))
	}
}

func TestConsole_GetHeightWidth_Error(t *testing.T) {
	// Create a console with a file that will cause pty.Getsize to fail
	ptmx, pts, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = ptmx.Close()
		_ = pts.Close()
	}()

	// Close ptmx to make Getsize fail
	_ = ptmx.Close()

	c := &console{file: ptmx}
	height, width := c.GetHeightWidth()

	// Should return default values when Getsize fails
	if height != MinHeight {
		t.Errorf("GetHeightWidth() height = %d, want %d", height, MinHeight)
	}
	if width != MinWidth {
		t.Errorf("GetHeightWidth() width = %d, want %d", width, MinWidth)
	}
}

func TestConstants(t *testing.T) {
	if MinWidth != 80 {
		t.Errorf("MinWidth = %d, want 80", MinWidth)
	}
	if MinHeight != 10 {
		t.Errorf("MinHeight = %d, want 10", MinHeight)
	}
	if Save != "\0337" {
		t.Errorf("Save = %q, want \\0337", Save)
	}
	if Restore != "\0338" {
		t.Errorf("Restore = %q, want \\0338", Restore)
	}
}
