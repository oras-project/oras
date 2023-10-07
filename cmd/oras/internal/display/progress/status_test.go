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
	"testing"
	"time"

	"oras.land/oras/cmd/oras/internal/display/console"
)

func Test_status_String(t *testing.T) {
	// zero status and progress
	s := newStatus()
	if status, progress := s.String(console.MinWidth); status != zeroStatus || progress != zeroProgress {
		t.Errorf("status.String() = %v, %v, want %v, %v", status, progress, zeroStatus, zeroProgress)
	}
}

func Test_status_durationString(t *testing.T) {
	// zero duration
	s := newStatus()
	if d := s.durationString(); d != zeroDuration {
		t.Errorf("status.durationString() = %v, want %v", d, zeroDuration)
	}

	// not ended
	s.startTime = time.Now().Add(-time.Second)
	if d := s.durationString(); d == zeroDuration {
		t.Errorf("status.durationString() = %v, want not %v", d, zeroDuration)
	}

	// ended: 61 seconds
	s.startTime = time.Now()
	s.endTime = s.startTime.Add(61 * time.Second)
	if d := s.durationString(); d != "1m1s" {
		t.Errorf("status.durationString() = %v, want %v", d, "1m1s")
	}

	// ended: 1001 Microsecond
	s.startTime = time.Now()
	s.endTime = s.startTime.Add(1001 * time.Microsecond)
	if d := s.durationString(); d != "1ms" {
		t.Errorf("status.durationString() = %v, want %v", d, "1ms")
	}

	// ended: 1001 Nanosecond
	s.startTime = time.Now()
	s.endTime = s.startTime.Add(1001 * time.Nanosecond)
	if d := s.durationString(); d != "1µs" {
		t.Errorf("status.durationString() = %v, want %v", d, "1µs")
	}
}
