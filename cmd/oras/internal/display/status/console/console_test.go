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
	containerd "github.com/containerd/console"
	"testing"

	"oras.land/oras/internal/testutils"
)

func givenConsole(t *testing.T) (Console, containerd.Console) {
	pty, _, err := containerd.NewPty()
	if err != nil {
		t.Fatal(err)
	}
	return &console{
		Console: pty,
	}, pty
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
	_, device, err := testutils.NewPty()
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()

	c, err := NewConsole(device)
	if err != nil {
		t.Fatal(err)
	}

	s, err := c.Size()
	if err != nil {
		t.Errorf("Unexpect error %v", err)
	}
	if s.Height != 10 {
		t.Errorf("Expected height 10 got %d", s.Height)
	}
	if s.Width != 80 {
		t.Errorf("Expected height 80 got %d", s.Width)
	}
}

func TestConsole_Size(t *testing.T) {
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

func TestConsole_GetHeightWidth(t *testing.T) {

}

func TestConsole_NewRow(t *testing.T) {

}

func TestConsole_OutputTo(t *testing.T) {

}

func TestConsole_Restore(t *testing.T) {

}
