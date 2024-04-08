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
	zeroStatus   = "loading status..."
	zeroDigest   = "  └─ loading digest..."
)

// status is used as message to update progress view.
type status struct {
	done           bool // done is true when the end time is set
	prompt         string
	descriptor     ocispec.Descriptor
	offset         int64
	total          humanize.Bytes
	lastOffset     int64
	lastRenderTime time.Time

	startTime time.Time
	endTime   time.Time
	mark      spinner
	lock      sync.Mutex
}

// newStatus generates a base empty status.
func newStatus() *status {
	return &status{
		offset:         -1,
		total:          humanize.ToBytes(0),
		lastRenderTime: time.Now(),
	}
}

// NewStatus generates a status.
func NewStatus(prompt string, descriptor ocispec.Descriptor, offset int64) *status {
	return &status{
		prompt:     prompt,
		descriptor: descriptor,
		offset:     offset,
	}
}

// StartTiming starts timing.
func StartTiming() *status {
	return &status{
		offset:    -1,
		startTime: time.Now(),
	}
}

// EndTiming ends timing and set status to done.
func EndTiming() *status {
	return &status{
		offset:  -1,
		endTime: time.Now(),
	}
}

func (s *status) isZero() bool {
	return s.offset < 0 && s.startTime.IsZero() && s.endTime.IsZero()
}

// String returns human-readable TTY strings of the status.
func (s *status) String(width int) (string, string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.isZero() {
		return zeroStatus, zeroDigest
	}
	// todo: doesn't support multiline prompt
	total := uint64(s.descriptor.Size)
	var percent float64

	name := s.descriptor.Annotations["org.opencontainers.image.title"]
	if name == "" {
		name = s.descriptor.MediaType
	}

	// format:  [left--------------------------------------------][margin][right---------------------------------]
	//          mark(1) bar(22) speed(8) action(<=11) name(<=126)        size_per_size(<=13) percent(8) time(>=6)
	//           └─ digest(72)
	var offset string
	switch s.done {
	case true: // 100%, show exact size
		offset = fmt.Sprint(s.total.Size)
		percent = 1
	default: // 0% ~ 99%, show 2-digit precision
		if total != 0 && s.offset >= 0 {
			// percentage calculatable
			percent = float64(s.offset) / float64(total)
		}
		offset = fmt.Sprintf("%.2f", humanize.RoundTo(s.total.Size*percent))
	}
	right := fmt.Sprintf(" %s/%s %6.2f%% %6s", offset, s.total, percent*100, s.durationString())
	lenRight := utf8.RuneCountInString(right)

	var left string
	lenLeft := 0
	if !s.done {
		lenBar := int(percent * barLength)
		bar := fmt.Sprintf("[%s%s]", aec.Inverse.Apply(strings.Repeat(" ", lenBar)), strings.Repeat(".", barLength-lenBar))
		speed := s.calculateSpeed()
		left = fmt.Sprintf("%c %s(%*s/s) %s %s", s.mark.symbol(), bar, speedLength, speed, s.prompt, name)
		// bar + wrapper(2) + space(1) + speed + "/s"(2) + wrapper(2) = len(bar) + len(speed) + 7
		lenLeft = barLength + speedLength + 7
	} else {
		left = fmt.Sprintf("✓ %s %s", s.prompt, name)
	}
	// mark(1) + space(1) + prompt + space(1) + name = len(prompt) + len(name) + 3
	lenLeft += utf8.RuneCountInString(s.prompt) + utf8.RuneCountInString(name) + 3

	lenMargin := width - lenLeft - lenRight
	if lenMargin < 0 {
		// hide partial name with one space left
		left = left[:len(left)+lenMargin-1] + "."
		lenMargin = 0
	}
	return fmt.Sprintf("%s%s%s", left, strings.Repeat(" ", lenMargin), right), fmt.Sprintf("  └─ %s", s.descriptor.Digest.String())
}

// calculateSpeed calculates the speed of the progress and update last status.
// caller must hold the lock.
func (s *status) calculateSpeed() humanize.Bytes {
	now := time.Now()
	if s.lastRenderTime.IsZero() {
		s.lastRenderTime = s.startTime
	}
	secondsTaken := now.Sub(s.lastRenderTime).Seconds()
	if secondsTaken == 0 {
		secondsTaken = bufFlushDuration.Seconds()
	}
	bytes := float64(s.offset - s.lastOffset)

	s.lastOffset = s.offset
	s.lastRenderTime = now

	return humanize.ToBytes(int64(bytes / secondsTaken))
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

	switch {
	case d > time.Second:
		d = d.Round(time.Second)
	case d > time.Millisecond:
		d = d.Round(time.Millisecond)
	default:
		d = d.Round(time.Microsecond)
	}
	return d.String()
}

// Update updates a status.
func (s *status) Update(n *status) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if n.offset >= 0 {
		s.offset = n.offset
		if n.descriptor.Size != s.descriptor.Size {
			s.total = humanize.ToBytes(n.descriptor.Size)
		}
		s.descriptor = n.descriptor
	}
	if n.prompt != "" {
		s.prompt = n.prompt
	}
	if !n.startTime.IsZero() {
		s.startTime = n.startTime
	}
	if !n.endTime.IsZero() {
		s.endTime = n.endTime
		s.done = true
	}
}
