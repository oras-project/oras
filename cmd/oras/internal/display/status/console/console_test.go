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
	"bytes"
	"errors"
	"os"
	"testing"
)

var errFake = errors.New("fake size error")

// fakeTerminal is an in-memory terminal used to exercise the cursor and escape
// code logic without a real terminal.
type fakeTerminal struct {
	bytes.Buffer
	height int
	width  int
	err    error
}

func (f *fakeTerminal) size() (height, width int, err error) {
	return f.height, f.width, f.err
}

func newFakeConsole(ft *fakeTerminal) *console {
	return &console{term: ft}
}

func TestNewConsole(t *testing.T) {
	if _, err := NewConsole(nil); err == nil {
		t.Error("expected error creating console from nil file")
	}
	c, err := NewConsole(os.Stdin)
	if err != nil {
		t.Errorf("unexpected error creating console: %v", err)
	}
	if c == nil {
		t.Error("expected a non-nil console")
	}
}

func TestConsole_GetHeightWidth(t *testing.T) {
	tests := []struct {
		name                  string
		ft                    *fakeTerminal
		wantHeight, wantWidth int
	}{
		{"size error falls back to minimum", &fakeTerminal{err: errFake}, MinHeight, MinWidth},
		{"below minimum is clamped", &fakeTerminal{height: 1, width: 1}, MinHeight, MinWidth},
		{"width below minimum is clamped", &fakeTerminal{height: 100, width: 1}, 100, MinWidth},
		{"height below minimum is clamped", &fakeTerminal{height: 1, width: 200}, MinHeight, 200},
		{"valid size is preserved", &fakeTerminal{height: 100, width: 200}, 100, 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newFakeConsole(tt.ft)
			gotHeight, gotWidth := c.GetHeightWidth()
			if gotHeight != tt.wantHeight || gotWidth != tt.wantWidth {
				t.Errorf("GetHeightWidth() = (%d, %d), want (%d, %d)", gotHeight, gotWidth, tt.wantHeight, tt.wantWidth)
			}
		})
	}
}

func TestConsole_Save(t *testing.T) {
	ft := &fakeTerminal{}
	c := newFakeConsole(ft)
	c.Save()
	if got, want := ft.String(), "\x1b[?25l\x1b7\x1b[0m"; got != want {
		t.Errorf("Save() = %q, want %q", got, want)
	}
}

func TestConsole_NewRow(t *testing.T) {
	ft := &fakeTerminal{}
	c := newFakeConsole(ft)
	c.NewRow()
	if got, want := ft.String(), "\x1b8\n\x1b7"; got != want {
		t.Errorf("NewRow() = %q, want %q", got, want)
	}
}

func TestConsole_OutputTo(t *testing.T) {
	ft := &fakeTerminal{}
	c := newFakeConsole(ft)
	c.OutputTo(1, "test string")
	if got, want := ft.String(), "\x1b8\x1b[1Ftest string\x1b[0m\n\x1b[0K"; got != want {
		t.Errorf("OutputTo() = %q, want %q", got, want)
	}
}

func TestConsole_Restore(t *testing.T) {
	ft := &fakeTerminal{}
	c := newFakeConsole(ft)
	c.Restore()
	if got, want := ft.String(), "\x1b8\x1b[0G\x1b[2K\x1b[?25h"; got != want {
		t.Errorf("Restore() = %q, want %q", got, want)
	}
}
