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

	"github.com/dustin/go-humanize"
	"github.com/morikuni/aec"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const BarMaxLength = 40

// status is used as message to update progress view.
type status struct {
	done       bool // done is true when the end time is set
	prompt     string
	descriptor ocispec.Descriptor
	offset     int64
	startTime  time.Time
	endTime    time.Time
	mark       spinner
	lock       sync.RWMutex
}

// newStatus generates a base empty status
func newStatus() *status {
	return &status{
		offset: -1,
	}
}

// NewStatus generates a status.
func NewStatus(prompt string, descriptor ocispec.Descriptor, offset uint64) *status {
	return &status{
		prompt:     prompt,
		descriptor: descriptor,
		offset:     int64(offset),
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
	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.isZero() {
		return "loading status...", "loading progress..."
	}
	// todo: doesn't support multiline prompt
	total := uint64(s.descriptor.Size)
	percent := float64(s.offset) / float64(total)

	name := s.descriptor.Annotations["org.opencontainers.image.title"]
	if name == "" {
		name = s.descriptor.MediaType
	}

	// format:  [left-------------------------------][margin][right-----------------------------]
	//          mark(1) bar(42) action(<10) name(126)        size_per_size(19) percent(8) time(8)
	//           └─ digest(72)
	var left string
	var lenLeft int
	if !s.done {
		lenBar := int(percent * BarMaxLength)
		bar := fmt.Sprintf("[%s%s]", aec.Inverse.Apply(strings.Repeat(" ", lenBar)), strings.Repeat(".", BarMaxLength-lenBar))
		mark := s.mark.symbol()
		left = fmt.Sprintf("%c %s %s %s", mark, bar, s.prompt, name)
		// bar + wrapper(2) + space(1) = len(bar) + 3
		lenLeft = BarMaxLength + 3
	} else {
		left = fmt.Sprintf("√ %s %s", s.prompt, name)
	}
	// mark(1) + space(1) + prompt + space(1) + name = len(prompt) + len(name) + 3
	lenLeft += 3 + utf8.RuneCountInString(s.prompt) + utf8.RuneCountInString(name)

	right := fmt.Sprintf(" %s/%s %6.2f%% %s", humanize.Bytes(uint64(s.offset)), humanize.Bytes(total), percent*100, s.durationString())
	lenRight := utf8.RuneCountInString(right)
	lenMargin := width - lenLeft - lenRight
	if lenMargin < 0 {
		// hide partial name with one space left
		left = left[:len(left)+lenMargin-1] + "."
		lenMargin = 0
	}
	return fmt.Sprintf("%s%s%s", left, strings.Repeat(" ", lenMargin), right), fmt.Sprintf("  └─ %s", s.descriptor.Digest.String())
}

// durationString returns a viewable TTY string of the status with duration.
func (s *status) durationString() string {
	if s.startTime.IsZero() {
		return "0ms"
	}

	var d time.Duration
	if s.endTime.IsZero() {
		d = time.Since(s.startTime)
	} else {
		d = s.endTime.Sub(s.startTime)
	}

	switch {
	case d > time.Minute:
		d = d.Round(time.Second)
	case d > time.Second:
		d = d.Round(100 * time.Millisecond)
	case d > time.Millisecond:
		d = d.Round(time.Millisecond)
	default:
		d = d.Round(10 * time.Nanosecond)
	}
	return d.String()
}

// Update updates a status.
func (s *status) Update(n *status) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if n.offset >= 0 {
		s.offset = n.offset
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
