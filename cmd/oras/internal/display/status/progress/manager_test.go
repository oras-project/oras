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
	"errors"
	"regexp"
	"sync"
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

func Test_manager_Track_concurrentClose(t *testing.T) {
	desc := ocispec.Descriptor{
		MediaType: "application/octet-stream",
		Size:      1,
		Digest:    "sha256:6e340b9cffb37a989ca544e6bb780a2c78901d3fb33738768511a30617afa01d",
	}
	for range 100 {
		m := newManager(newMockConsole(80, 24), nil).(*manager)
		trackErr := make(chan error, 1)
		go func() {
			tracker, err := m.Track(desc)
			if err == nil {
				err = tracker.Close()
			}
			trackErr <- err
		}()
		if err := m.Close(); err != nil {
			t.Fatalf("manager.Close() error = %v, wantErr nil", err)
		}
		m.lock.RLock()
		statuses := m.status
		m.lock.RUnlock()

		err := <-trackErr
		if errors.Is(err, errManagerStopped) {
			continue
		}
		if err != nil {
			t.Fatalf("manager.Track() error = %v, wantErr nil or %v", err, errManagerStopped)
		}
		// a successful Track must be drained before Close returns
		if len(statuses) != 1 {
			t.Fatal("manager.Track() succeeded after manager.Close() returned")
		}
		s := statuses[0]
		s.lock.RLock()
		done := s.done
		s.lock.RUnlock()
		if !done {
			t.Fatal("tracked status is not drained when manager.Close() returns")
		}
	}
}

func Test_manager_Close_concurrent(t *testing.T) {
	for range 100 {
		m := newManager(newMockConsole(80, 24), nil)
		errs := make(chan error, 2)
		var wg sync.WaitGroup
		for range 2 {
			wg.Go(func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("manager.Close() panicked: %v", r)
					}
				}()
				errs <- m.Close()
			})
		}
		wg.Wait()
		close(errs)
		if t.Failed() {
			return
		}

		var stopped int
		for err := range errs {
			switch {
			case err == nil:
			case errors.Is(err, errManagerStopped):
				stopped++
			default:
				t.Fatalf("manager.Close() error = %v, wantErr nil or %v", err, errManagerStopped)
			}
		}
		if stopped != 1 {
			t.Fatalf("manager.Close() returned %v %d times, want 1", errManagerStopped, stopped)
		}
	}
}
