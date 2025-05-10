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

	containerd "github.com/containerd/console"
	"github.com/morikuni/aec"
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

// Console is a wrapper around containerd's Console and ANSI escape codes.
type Console interface {
	containerd.Console
	GetHeightWidth() (height, width int)
	Save()
	NewRow()
	OutputTo(upCnt uint, str string)
	Restore()
}

type console struct {
	containerd.Console
}

// NewConsole generates a console from a file.
func NewConsole(f *os.File) (Console, error) {
	c, err := containerd.ConsoleFromFile(f)
	if err != nil {
		return nil, err
	}
	return &console{c}, nil
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
