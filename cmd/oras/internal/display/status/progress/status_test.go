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
	"reflect"
	"regexp"
	"testing"
	"time"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/status/progress/humanize"
)

func Test_status_Render(t *testing.T) {
	escRegexp := regexp.MustCompile("\x1b\\[[0-9]+m")
	equal := func(got, want [2]string) bool {
		noColor := [2]string{
			escRegexp.ReplaceAllString(got[0], ""),
			got[1],
		}
		return noColor == want
	}
	desc := ocispec.Descriptor{
		MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
		Size:      1234567890,
		Digest:    "sha256:c775e7b757ede630cd0aa1113bd102661ab38829ca52a6422ab782862f268646",
		Annotations: map[string]string{
			"org.opencontainers.image.title": "hello.bin",
		},
	}
	tests := []struct {
		name   string
		status func() *status // constructor required to work the time.Now() around
		width  int
		want   [2]string
	}{
		{
			name: "default status",
			status: func() *status {
				return newStatus(desc)
			},
			width: 80,
			want: [2]string{
				"⠋ [....................](   0  B/s)  hello.bin          -/1.15 GB   0.00%     0s",
				"  └─ sha256:c775e7b757ede630cd0aa1113bd102661ab38829ca52a6422ab782862f268646    ",
			},
		},
		{
			name: "operation in progress",
			status: func() *status {
				return &status{
					text:       "Test",
					startTime:  time.Now().Add(-time.Second * 100),
					descriptor: desc,
					offset:     123456789,
					total:      humanize.ToBytes(desc.Size),
					speed:      newSpeedWindow(10),
				}
			},
			width: 80,
			want: [2]string{
				"⠋ [  ..................](   0  B/s) Test hello.bin   0.12/1.15 GB  10.00%  1m40s",
				"  └─ sha256:c775e7b757ede630cd0aa1113bd102661ab38829ca52a6422ab782862f268646    ",
			},
		},
		{
			name: "operation succeeded",
			status: func() *status {
				return &status{
					done:       true,
					text:       "Tested",
					startTime:  time.Now().Add(-time.Second * 100),
					endTime:    time.Now(),
					descriptor: desc,
					offset:     1234567890,
					total:      humanize.ToBytes(desc.Size),
					speed:      newSpeedWindow(10),
				}
			},
			width: 80,
			want: [2]string{
				"✓ Tested hello.bin                                   1.15/1.15 GB 100.00%  1m40s",
				"  └─ sha256:c775e7b757ede630cd0aa1113bd102661ab38829ca52a6422ab782862f268646    ",
			},
		},
		{
			name: "operation failed",
			status: func() *status {
				return &status{
					text:       "Test",
					err:        errors.New("test error"),
					startTime:  time.Now().Add(-time.Second * 100),
					descriptor: desc,
					offset:     123456789,
					total:      humanize.ToBytes(desc.Size),
					speed:      newSpeedWindow(10),
				}
			},
			width: 80,
			want: [2]string{
				"✗ [  ..................](   0  B/s) Test hello.bin   0.12/1.15 GB  10.00%  1m40s",
				"  └─ sha256:c775e7b757ede630cd0aa1113bd102661ab38829ca52a6422ab782862f268646    ",
			},
		},
		{
			name: "long status text in progress",
			status: func() *status {
				return &status{
					text:       "Longer test",
					startTime:  time.Now().Add(-time.Second * 100),
					descriptor: desc,
					offset:     123456789,
					total:      humanize.ToBytes(desc.Size),
					speed:      newSpeedWindow(10),
				}
			},
			width: 80,
			want: [2]string{
				"⠋ [  ..................](   0  B/s) Longer test hel. 0.12/1.15 GB  10.00%  1m40s",
				"  └─ sha256:c775e7b757ede630cd0aa1113bd102661ab38829ca52a6422ab782862f268646    ",
			},
		},
		{
			name: "object with no name in progress",
			status: func() *status {
				plainDesc := desc
				plainDesc.Annotations = nil
				return &status{
					text:       "Test",
					startTime:  time.Now().Add(-time.Second * 100),
					descriptor: plainDesc,
					offset:     123456789,
					total:      humanize.ToBytes(plainDesc.Size),
					speed:      newSpeedWindow(10),
				}
			},
			width: 80,
			want: [2]string{
				"⠋ [  ..................](   0  B/s) Test applicatio. 0.12/1.15 GB  10.00%  1m40s",
				"  └─ sha256:c775e7b757ede630cd0aa1113bd102661ab38829ca52a6422ab782862f268646    ",
			},
		},
		{
			name: "object with zero size in progress",
			status: func() *status {
				zeroDesc := ocispec.Descriptor{
					MediaType: "text/plain",
					Size:      0,
					Digest:    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				}
				return &status{
					text:       "Test",
					startTime:  time.Now().Add(-time.Second * 100),
					descriptor: zeroDesc,
					total:      humanize.ToBytes(zeroDesc.Size),
					speed:      newSpeedWindow(10),
				}
			},
			width: 80,
			want: [2]string{
				"⠋ [                    ](   0  B/s) Test text/plain        0/0  B 100.00%  1m40s",
				"  └─ sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855    ",
			},
		},
		{
			name: "long status text succeeded",
			status: func() *status {
				return &status{
					done:       true,
					text:       "Long Long Long Long Long Long Long Long test",
					startTime:  time.Now().Add(-time.Second * 100),
					endTime:    time.Now(),
					descriptor: desc,
					offset:     1234567890,
					total:      humanize.ToBytes(desc.Size),
					speed:      newSpeedWindow(10),
				}
			},
			width: 80,
			want: [2]string{
				"✓ Long Long Long Long Long Long Long Long test hell. 1.15/1.15 GB 100.00%  1m40s",
				"  └─ sha256:c775e7b757ede630cd0aa1113bd102661ab38829ca52a6422ab782862f268646    ",
			},
		},
		{
			name: "object with no name succeeded",
			status: func() *status {
				plainDesc := desc
				plainDesc.Annotations = nil
				return &status{
					done:       true,
					text:       "Tested",
					startTime:  time.Now().Add(-time.Second * 100),
					endTime:    time.Now(),
					descriptor: plainDesc,
					offset:     1234567890,
					total:      humanize.ToBytes(plainDesc.Size),
					speed:      newSpeedWindow(10),
				}
			},
			width: 80,
			want: [2]string{
				"✓ Tested application/vnd.docker.image.rootfs.diff.t. 1.15/1.15 GB 100.00%  1m40s",
				"  └─ sha256:c775e7b757ede630cd0aa1113bd102661ab38829ca52a6422ab782862f268646    ",
			},
		},
		{
			name: "object with zero size succeeded",
			status: func() *status {
				zeroDesc := ocispec.Descriptor{
					MediaType: "text/plain",
					Size:      0,
					Digest:    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				}
				return &status{
					done:       true,
					text:       "Tested",
					startTime:  time.Now().Add(-time.Second * 100),
					endTime:    time.Now(),
					descriptor: zeroDesc,
					total:      humanize.ToBytes(zeroDesc.Size),
					speed:      newSpeedWindow(10),
				}
			},
			width: 80,
			want: [2]string{
				"✓ Tested text/plain                                        0/0  B 100.00%  1m40s",
				"  └─ sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855    ",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.status()
			if got := s.Render(tt.width); !equal(got, tt.want) {
				t.Errorf("status.Render() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_status_durationString(t *testing.T) {
	tests := []struct {
		name string
		s    *status
		want string
	}{
		{
			name: "duration in hours",
			s: &status{
				startTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				endTime:   time.Date(2021, 1, 1, 2, 1, 10, 0, time.UTC),
			},
			want: "2h1m10s",
		},
		{
			name: "duration in minutes",
			s: &status{
				startTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				endTime:   time.Date(2021, 1, 1, 0, 1, 10, 0, time.UTC),
			},
			want: "1m10s",
		},
		{
			name: "duration in seconds",
			s: &status{
				startTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				endTime:   time.Date(2021, 1, 1, 0, 0, 10, 0, time.UTC),
			},
			want: "10s",
		},
		{
			name: "duration in milliseconds",
			s: &status{
				startTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				endTime:   time.Date(2021, 1, 1, 0, 0, 0, 10000000, time.UTC),
			},
			want: "10ms",
		},
		{
			name: "duration in microseconds",
			s: &status{
				startTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				endTime:   time.Date(2021, 1, 1, 0, 0, 0, 10000, time.UTC),
			},
			want: "10µs",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.durationString(); got != tt.want {
				t.Errorf("status.durationString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_updateStatus(t *testing.T) {
	desc := ocispec.Descriptor{
		MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
		Size:      1234567890,
		Digest:    "sha256:c775e7b757ede630cd0aa1113bd102661ab38829ca52a6422ab782862f268646",
		Annotations: map[string]string{
			"org.opencontainers.image.title": "hello.bin",
		},
	}
	equal := func(s, t *status) bool {
		// speed is not compared
		return s.done == t.done &&
			s.err == t.err &&
			s.text == t.text &&
			s.startTime.Equal(t.startTime) &&
			s.endTime.Equal(t.endTime) &&
			reflect.DeepEqual(s.descriptor, t.descriptor) &&
			s.offset == t.offset &&
			reflect.DeepEqual(s.total, t.total)
	}
	errTest := errors.New("test error")
	tests := []struct {
		name   string
		setup  statusUpdate
		update statusUpdate
		want   *status
		check  func(*testing.T, *status) // custom check instead of want
	}{
		{
			name:   "updateStatusMessage",
			update: updateStatusMessage("Test", 42),
			want: &status{
				text:       "Test",
				descriptor: desc,
				offset:     42,
				total:      humanize.ToBytes(desc.Size),
			},
		},
		{
			name:   "updateStatusMessage (text only)",
			setup:  updateStatusMessage("foo", 42),
			update: updateStatusMessage("bar", -1),
			want: &status{
				text:       "bar",
				descriptor: desc,
				offset:     42,
				total:      humanize.ToBytes(desc.Size),
			},
		},
		{
			name:   "updateStatusStartTime",
			update: updateStatusStartTime(),
			check: func(t *testing.T, s *status) {
				if s.startTime.IsZero() {
					t.Errorf("updateStatusStartTime() = %v, want non-zero", s.startTime)
				}
				if s.speed.next != 1 {
					t.Errorf("updateStatusStartTime() did not add a speed sample")
				}
			},
		},
		{
			name:   "updateStatusEndTime",
			update: updateStatusEndTime(),
			check: func(t *testing.T, s *status) {
				if s.endTime.IsZero() {
					t.Errorf("updateStatusEndTime() = %v, want non-zero", s.endTime)
				}
				if !s.done {
					t.Errorf("updateStatusEndTime() did not set done")
				}
			},
		},
		{
			name:   "updateStatusEndTime with error",
			setup:  updateStatusError(errTest),
			update: updateStatusEndTime(),
			check: func(t *testing.T, s *status) {
				if s.endTime.IsZero() {
					t.Errorf("updateStatusEndTime() = %v, want non-zero", s.endTime)
				}
				if s.done {
					t.Errorf("updateStatusEndTime() set done")
				}
			},
		},
		{
			name:   "updateStatusError",
			update: updateStatusError(errTest),
			want: &status{
				err:        errTest,
				descriptor: desc,
				offset:     -1,
				total:      humanize.ToBytes(desc.Size),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newStatus(desc)
			if tt.setup != nil {
				tt.setup(got)
			}
			if tt.update(got); tt.check != nil {
				tt.check(t, got)
			} else if !equal(got, tt.want) {
				t.Errorf("updateStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}
