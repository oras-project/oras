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
	"errors"
	"io"
	"os"

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

// Console is a wrapper around a terminal and ANSI escape codes.
type Console interface {
	GetHeightWidth() (height, width int)
	Save()
	NewRow()
	OutputTo(upCnt uint, str string)
	Restore()
}

// terminal is the minimal backend that a Console renders to. It decouples the
// cursor and escape-code logic from the underlying terminal implementation so
// that the backing library (currently golang.org/x/term) can be swapped or
// extended without touching callers.
type terminal interface {
	io.Writer
	// size returns the height and width of the terminal in cells.
	size() (height, width int, err error)
}

// fileTerminal is a terminal backed by an *os.File, using golang.org/x/term
// for size queries.
type fileTerminal struct {
	*os.File
}

func (t *fileTerminal) size() (height, width int, err error) {
	width, height, err = term.GetSize(int(t.Fd()))
	return height, width, err
}

type console struct {
	term terminal
}

// NewConsole generates a console from a file.
func NewConsole(f *os.File) (Console, error) {
	if f == nil {
		return nil, errors.New("cannot create console from nil file")
	}
	return &console{term: &fileTerminal{f}}, nil
}

// GetHeightWidth returns the height and width of the console.
// If the console size cannot be determined, returns a default value of 80x10.
func (c *console) GetHeightWidth() (height, width int) {
	height, width, err := c.term.size()
	if err != nil {
		return MinHeight, MinWidth
	}
	if height < MinHeight {
		height = MinHeight
	}
	if width < MinWidth {
		width = MinWidth
	}
	return height, width
}

// Save saves the current cursor position.
func (c *console) Save() {
	_, _ = c.term.Write([]byte(aec.Hide.Apply(Save)))
}

// NewRow allocates a horizontal space to the output area with scroll if needed.
func (c *console) NewRow() {
	_, _ = c.term.Write([]byte(Restore))
	_, _ = c.term.Write([]byte("\n"))
	_, _ = c.term.Write([]byte(Save))
}

// OutputTo outputs a string to a specific line.
func (c *console) OutputTo(upCnt uint, str string) {
	_, _ = c.term.Write([]byte(Restore))
	_, _ = c.term.Write([]byte(aec.PreviousLine(upCnt).Apply(str)))
	_, _ = c.term.Write([]byte("\n"))
	_, _ = c.term.Write([]byte(aec.EraseLine(aec.EraseModes.Tail).String()))
}

// Restore restores the saved cursor position.
func (c *console) Restore() {
	// cannot use aec.Restore since DEC has better compatibility than SCO
	_, _ = c.term.Write([]byte(Restore))
	_, _ = c.term.Write([]byte(aec.Column(0).
		With(aec.EraseLine(aec.EraseModes.All)).
		With(aec.Show).String()))
}
