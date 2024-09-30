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
)

func TestConsole_New(t *testing.T) {
	mockFile, err := os.OpenFile(os.DevNull, os.O_RDWR, 0666)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	sut, err := NewConsole(mockFile)
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}

	if err = sut.Resize(containerd.WinSize{}); err != nil {
		t.Errorf("Unexpected erro for Resize: %v", err)
	}
	if err = sut.ResizeFrom(nil); err != nil {
		t.Errorf("Unexpected erro for Resize: %v", err)
	}
	if err = sut.SetRaw(); err != nil {
		t.Errorf("Unexpected erro for Resize: %v", err)
	}
	if err = sut.DisableEcho(); err != nil {
		t.Errorf("Unexpected erro for Resize: %v", err)
	}
	if err = sut.Reset(); err != nil {
		t.Errorf("Unexpected erro for Resize: %v", err)
	}
	windowSize, _ := sut.Size()
	if windowSize.Height != 24 {
		t.Errorf("Expected size 24 actual %d", windowSize.Height)
	}
	if windowSize.Width != 80 {
		t.Errorf("Expected size 80 actual %d", windowSize.Width)
	}
	h, w := sut.GetHeightWidth()
	if h != 24 {
		t.Errorf("Expected size 24 actual %d", h)
	}
	if w != 80 {
		t.Errorf("Expected size 80 actual %d", w)
	}
	if sut.Fd() != os.Stderr.Fd() {
		t.Errorf("Expected size %d actual %d", sut.Fd(), os.Stderr.Fd())
	}
	if sut.Name() != os.DevNull {
		t.Errorf("Expected size %s actual %s", sut.Name(), os.DevNull)
	}
	sut.OutputTo(0, "ignored")
	sut.Restore()
}
