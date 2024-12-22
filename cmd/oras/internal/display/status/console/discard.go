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
)

type discardConsole struct {
	*os.File
}

// NewDiscardConsole create a console that does not output.
func NewDiscardConsole(f *os.File) Console {
	dc := discardConsole{
		File: f,
	}
	return &dc
}

// Fd returns its file descriptor
func (mc *discardConsole) Fd() uintptr {
	return os.Stderr.Fd()
}

// Name returns its file name
func (mc *discardConsole) Name() string {
	return mc.File.Name()
}

// Resize ignored
func (mc *discardConsole) Resize(_ containerd.WinSize) error {
	return nil
}

// ResizeFrom ignored
func (mc *discardConsole) ResizeFrom(containerd.Console) error {
	return nil
}

// SetRaw ignored
func (mc *discardConsole) SetRaw() error {
	return nil
}

// DisableEcho ignored
func (mc *discardConsole) DisableEcho() error {
	return nil
}

// Reset ignored
func (mc *discardConsole) Reset() error {
	return nil
}

// Size return default size
func (mc *discardConsole) Size() (containerd.WinSize, error) {
	ws := containerd.WinSize{
		Width:  80,
		Height: 24,
	}
	return ws, nil
}

// GetHeightWidth returns the width and height of the console.
func (mc *discardConsole) GetHeightWidth() (height, width int) {
	windowSize, _ := mc.Size()
	return int(windowSize.Height), int(windowSize.Width)
}

// Save ignored
func (mc *discardConsole) Save() {
}

// NewRow ignored
func (mc *discardConsole) NewRow() {
}

// OutputTo ignored
func (mc *discardConsole) OutputTo(_ uint, _ string) {
}

// Restore ignored
func (mc *discardConsole) Restore() {
}
