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
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/morikuni/aec"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/status/progress/humanize"
)

const (
	barLength    = 20
	speedLength  = 7    // speed_size(4) + space(1) + speed_unit(2)
	zeroDuration = "0s" // default zero value of time.Duration.String()
)

var (
	spinnerColor  = aec.LightYellowF
	doneMarkColor = aec.LightGreenF
	progressColor = aec.LightBlueB
	failureColor  = aec.LightRedF
)

// status is the model to present the progress of an operation.
type status struct {
	lock sync.RWMutex
	done bool  // true if the operation is succeeded
	err  error // non-nil if the operation fails

	mark      spinner
	text      string
	startTime time.Time
	endTime   time.Time

	descriptor ocispec.Descriptor
	offset     int64
	total      humanize.Bytes
	speed      *speedWindow
}

// newStatus generates a base empty status.
func newStatus(desc ocispec.Descriptor) *status {
	return &status{
		descriptor: desc,
		offset:     -1,
		total:      humanize.ToBytes(desc.Size),
		speed:      newSpeedWindow(framePerSecond),
	}
}

// Render returns human-readable TTY strings of the status.
// Format:
//
//	[left--------------------------------------------][margin][right---------------------------------]
//	mark(1) bar(22) speed(8) action(<=11) name(<=126)        size_per_size(<=13) percent(8) time(>=6)
//	 └─ digest(72)
func (s *status) Render(width int) [2]string {
	s.lock.RLock()
	defer s.lock.RUnlock()

	// obtain object name
	name := s.descriptor.Annotations[ocispec.AnnotationTitle]
	if name == "" {
		name = s.descriptor.MediaType
	}

	// calculate the progress percentage
	var offset string
	var percent float64
	if s.done {
		// 100%, show exact size
		offset = fmt.Sprint(s.total.Size)
		percent = 1
	} else if s.offset < 0 {
		// not started, show 0%
		offset = "-"
	} else if s.descriptor.Size == 0 {
		// 0 byte, show 100%
		offset = "0"
		percent = 1
	} else {
		// 0% ~ 99%, show 2-digit precision
		percent = float64(s.offset) / float64(s.descriptor.Size)
		offset = fmt.Sprintf("%.2f", humanize.RoundTo(s.total.Size*percent))
	}

	// render the left side of the primary line
	var left string
	lenLeft := 0 // manually calculate the string length due to the color escape sequence
	if s.done {
		left = fmt.Sprintf("%s %s %s", doneMarkColor.Apply("✓"), s.text, name)
	} else {
		var mark string
		if s.err == nil {
			mark = spinnerColor.Apply(string(s.mark.symbol()))
		} else {
			mark = failureColor.Apply("✗")
		}
		lenBar := int(percent * barLength)
		speed := s.calculateSpeed()
		left = fmt.Sprintf("%s [%s%s](%*s/s) %s %s", mark,
			progressColor.Apply(strings.Repeat(" ", lenBar)), strings.Repeat(".", barLength-lenBar),
			speedLength, speed, s.text, name)
		// bar + wrapper(2) + space(1) + speed + "/s"(2) + wrapper(2) = len(bar) + len(speed) + 7
		lenLeft = barLength + speedLength + 7
	}
	// mark(1) + space(1) + prompt + space(1) + name = len(prompt) + len(name) + 3
	lenLeft += utf8.RuneCountInString(s.text) + utf8.RuneCountInString(name) + 3

	// render the right side of the primary line
	right := fmt.Sprintf(" %s/%s %6.2f%% %6s", offset, s.total, percent*100, s.durationString())
	lenRight := utf8.RuneCountInString(right)

	// render view
	lenMargin := width - lenLeft - lenRight
	if lenMargin < 0 {
		// hide partial name with one space left
		left = left[:len(left)+lenMargin-1] + "."
		lenMargin = 0
	}
	var padding string
	if paddingLen := width - len(s.descriptor.Digest) - 5; paddingLen > 0 {
		padding = strings.Repeat(" ", paddingLen)
	}
	return [2]string{
		fmt.Sprintf("%s%s%s", left, strings.Repeat(" ", lenMargin), right),
		fmt.Sprintf("  └─ %s%s", s.descriptor.Digest, padding),
	}
}

// calculateSpeed calculates the speed of the progress and update last status.
// caller must hold the lock.
func (s *status) calculateSpeed() humanize.Bytes {
	if s.offset < 0 {
		// not started
		return humanize.ToBytes(0)
	}
	s.speed.Add(time.Now(), s.offset)
	return humanize.ToBytes(int64(s.speed.Mean()))
}

// durationString returns a viewable TTY string of the status with duration.
func (s *status) durationString() string {
	if s.startTime.IsZero() {
		return zeroDuration
	}

	var d time.Duration
	if s.endTime.IsZero() {
		d = time.Since(s.startTime)
	} else {
		d = s.endTime.Sub(s.startTime)
	}

	return humanize.FormatDuration(d)
}

// statusUpdate is a function to update the status.
type statusUpdate func(*status)

// updateStatusMessage returns a statusUpdate to update the status message.
// Optionally, it can update the offset of the status.
func updateStatusMessage(text string, offset int64) statusUpdate {
	return func(s *status) {
		s.lock.Lock()
		defer s.lock.Unlock()

		s.text = text
		if offset >= 0 {
			s.offset = offset
		}
	}
}

// updateStatusStartTime returns a statusUpdate to update the status start time.
func updateStatusStartTime() statusUpdate {
	return func(s *status) {
		s.lock.Lock()
		defer s.lock.Unlock()

		s.startTime = time.Now()
		s.speed.Add(s.startTime, 0)
	}
}

// updateStatusEndTime returns a statusUpdate to update the status end time.
func updateStatusEndTime() statusUpdate {
	return func(s *status) {
		s.lock.Lock()
		defer s.lock.Unlock()

		s.endTime = time.Now()
		if s.err == nil {
			s.done = true
		}
	}
}

// updateStatusError returns a statusUpdate to update the status error.
func updateStatusError(err error) statusUpdate {
	return func(s *status) {
		s.lock.Lock()
		defer s.lock.Unlock()

		s.err = err
	}
}
