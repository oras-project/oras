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

package progress

import (
	"regexp"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/status/console"
	"oras.land/oras/internal/progress"
)

type mockConsole struct {
	console.Console

	view   []string
	height int
	width  int
}

func newMockConsole(width, height int) *mockConsole {
	return &mockConsole{
		height: height,
		width:  width,
	}
}

func (c *mockConsole) GetHeightWidth() (int, int) {
	return c.height, c.width
}

func (c *mockConsole) NewRow() {
	c.view = append(c.view, "")
}

func (c *mockConsole) OutputTo(upCnt uint, str string) {
	c.view[len(c.view)-int(upCnt)] = str
}

func (c *mockConsole) Restore() {}

func (c *mockConsole) Save() {}

func Test_manager(t *testing.T) {
	desc := ocispec.Descriptor{
		MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
		Size:      1234567890,
		Digest:    "sha256:c775e7b757ede630cd0aa1113bd102661ab38829ca52a6422ab782862f268646",
		Annotations: map[string]string{
			"org.opencontainers.image.title": "hello.bin",
		},
	}

	// simulate a console run
	c := newMockConsole(80, 24)
	m := newManager(c, map[progress.State]string{
		progress.StateExists: "Exists",
	})
	tracker, err := m.Track(desc)
	if err != nil {
		t.Fatalf("manager.Track() error = %v, wantErr nil", err)
	}
	if err = tracker.Update(progress.Status{
		State:  progress.StateExists,
		Offset: -1,
	}); err != nil {
		t.Errorf("tracker.Update() error = %v, wantErr nil", err)
	}
	if err := tracker.Close(); err != nil {
		t.Errorf("tracker.Close() error = %v, wantErr nil", err)
	}
	if err := m.Close(); err != nil {
		t.Errorf("manager.Close() error = %v, wantErr nil", err)
	}

	// verify the console output
	want := []string{
		"✓ Exists hello.bin                                   1.15/1.15 GB 100.00%     0s",
		"  └─ sha256:c775e7b757ede630cd0aa1113bd102661ab38829ca52a6422ab782862f268646    ",
	}
	if len(c.view) != len(want) {
		t.Errorf("console view length = %d, want %d", len(c.view), len(want))
	}
	escRegexp := regexp.MustCompile("\x1b\\[[0-9]+m")
	equal := func(got, want string) bool {
		return escRegexp.ReplaceAllString(got, "") == want
	}
	for i, v := range want {
		if !equal(c.view[i], v) {
			t.Errorf("console view[%d] = %q, want %q", i, c.view[i], v)
		}
	}
}
