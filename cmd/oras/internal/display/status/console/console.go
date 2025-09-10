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

	"github.com/creack/pty"
	"github.com/morikuni/aec"
	"golang.org/x/term"
)

const (
	// MinWidth is the minimal width of supported console.
	MinWidth = 80
	// MinHeight is the minimal height of supported console.
	MinHeight = 10
	// Save cannot use aec.Save since DEC has better compatibility than SCO
	Save = "\0337"
	// Restore cannot use aec.Restore since DEC has better compatibility than SCO
	Restore = "\0338"
)

// Console is a wrapper around PTY and ANSI escape codes.
type Console interface {
	Write(p []byte) (n int, err error)
	Size() (*WinSize, error)
	GetHeightWidth() (height, width int)
	Save()
	NewRow()
	OutputTo(upCnt uint, str string)
	Restore()
}

// WinSize represents the size of a terminal window.
type WinSize struct {
	Height uint16
	Width  uint16
}

type console struct {
	file *os.File
}

// NewConsole generates a console from a file.
func NewConsole(f *os.File) (Console, error) {
	if !term.IsTerminal(int(f.Fd())) {
		return nil, os.ErrInvalid
	}
	return &console{file: f}, nil
}

// Write writes data to the console.
func (c *console) Write(p []byte) (n int, err error) {
	return c.file.Write(p)
}

// Size returns the size of the console.
func (c *console) Size() (*WinSize, error) {
	rows, cols, err := pty.Getsize(c.file)
	if err != nil {
		return nil, err
	}
	return &WinSize{Height: uint16(rows), Width: uint16(cols)}, nil
}

// GetHeightWidth returns the width and height of the console.
// If the console size cannot be determined, returns a default value of 80x10.
func (c *console) GetHeightWidth() (height, width int) {
	windowSize, err := c.Size()
	if err != nil {
		return MinHeight, MinWidth
	}
	if windowSize.Height < MinHeight {
		windowSize.Height = MinHeight
	}
	if windowSize.Width < MinWidth {
		windowSize.Width = MinWidth
	}
	return int(windowSize.Height), int(windowSize.Width)
}

// Save saves the current cursor position.
func (c *console) Save() {
	_, _ = c.Write([]byte(aec.Hide.Apply(Save)))
}

// NewRow allocates a horizontal space to the output area with scroll if needed.
func (c *console) NewRow() {
	_, _ = c.Write([]byte(Restore))
	_, _ = c.Write([]byte("\n"))
	_, _ = c.Write([]byte(Save))
}

// OutputTo outputs a string to a specific line.
func (c *console) OutputTo(upCnt uint, str string) {
	_, _ = c.Write([]byte(Restore))
	_, _ = c.Write([]byte(aec.PreviousLine(upCnt).Apply(str)))
	_, _ = c.Write([]byte("\n"))
	_, _ = c.Write([]byte(aec.EraseLine(aec.EraseModes.Tail).String()))
}

// Restore restores the saved cursor position.
func (c *console) Restore() {
	// cannot use aec.Restore since DEC has better compatibility than SCO
	_, _ = c.Write([]byte(Restore))
	_, _ = c.Write([]byte(aec.Column(0).
		With(aec.EraseLine(aec.EraseModes.All)).
		With(aec.Show).String()))
}
