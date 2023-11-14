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

	"github.com/containerd/console"
	"github.com/morikuni/aec"
)

const (
	// MinWidth is the minimal width of supported console.
	MinWidth = 80
	// MinHeight is the minimal height of supported console.
	MinHeight = 10
	// cannot use aec.Save since DEC has better compatilibity than SCO
	Save = "\0337"
	// cannot use aec.Restore since DEC has better compatilibity than SCO
	Restore = "\0338"
)

// Console is a wrapper around containerd's console.Console and ANSI escape
// codes.
type Console struct {
	console.Console
}

// Size returns the width and height of the console.
// If the console size cannot be determined, returns a default value of 80x10.
func (c *Console) Size() (width, height int) {
	width = MinWidth
	height = MinHeight
	size, err := c.Console.Size()
	if err == nil {
		if size.Height > MinHeight {
			height = int(size.Height)
		}
		if size.Width > MinWidth {
			width = int(size.Width)
		}
	}
	return
}

// New generates a Console from a file.
func New(f *os.File) (*Console, error) {
	c, err := console.ConsoleFromFile(f)
	if err != nil {
		return nil, err
	}
	return &Console{c}, nil
}

// Save saves the current cursor position.
func (c *Console) Save() {
	_, _ = c.Write([]byte(aec.Hide.Apply(Save)))
}

// NewRow allocates a horizontal space to the output area with scroll if needed.
func (c *Console) NewRow() {
	_, _ = c.Write([]byte(Restore))
	_, _ = c.Write([]byte("\n"))
	_, _ = c.Write([]byte(Save))
}

// OutputTo outputs a string to a specific line.
func (c *Console) OutputTo(upCnt uint, str string) {
	_, _ = c.Write([]byte(Restore))
	_, _ = c.Write([]byte(aec.PreviousLine(upCnt).Apply(str)))
	_, _ = c.Write([]byte("\n"))
	_, _ = c.Write([]byte(aec.EraseLine(aec.EraseModes.Tail).String()))
}

// Restore restores the saved cursor position.
func (c *Console) Restore() {
	// cannot use aec.Restore since DEC has better compatilibity than SCO
	_, _ = c.Write([]byte(Restore))
	_, _ = c.Write([]byte(aec.Column(0).
		With(aec.EraseLine(aec.EraseModes.All)).
		With(aec.Show).String()))
}
