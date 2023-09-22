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
	"time"
	"unicode/utf8"

	"github.com/dustin/go-humanize"
	"github.com/morikuni/aec"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const BarMaxLength = 40

// status is used as message to update progress view.
type status struct {
	done       bool
	prompt     string
	descriptor ocispec.Descriptor
	offset     int64
	startTime  *time.Time
	endTime    *time.Time
	mark       mark
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
	now := time.Now()
	return &status{
		offset:    -1,
		startTime: &now,
	}
}

// EndTiming ends timing and set status to done.
func EndTiming() *status {
	now := time.Now()
	return &status{
		offset:  -1,
		endTime: &now,
	}
}

// String returns human-readable TTY strings of the status.
func (s *status) String(width int) (string, string) {
	if s == nil {
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
		mark := s.mark.GetMark()
		left = fmt.Sprintf("%c %s %s %s", mark, bar, s.prompt, name)
		// bar + wrapper(2) + space(1)
		lenLeft = BarMaxLength + 2 + 1
	} else {
		left = fmt.Sprintf("√ %s %s", s.prompt, name)
	}
	// mark(1) + space(1) + prompt+ space(1) + name
	lenLeft += 1 + 1 + utf8.RuneCountInString(s.prompt) + 1 + utf8.RuneCountInString(name)

	right := fmt.Sprintf(" %s/%s %6.2f%% %s", humanize.Bytes(uint64(s.offset)), humanize.Bytes(total), percent*100, s.DurationString())
	lenRight := utf8.RuneCountInString(right)
	lenMargin := width - lenLeft - lenRight
	if lenMargin < 0 {
		// hide partial name with one space left
		left = left[:len(left)+lenMargin-1] + "."
		lenMargin = 0
	}
	return fmt.Sprintf("%s%s%s", left, strings.Repeat(" ", lenMargin), right), fmt.Sprintf("  └─ %s", s.descriptor.Digest.String())
}

// DurationString returns a viewable TTY string of the status with duration.
func (s *status) DurationString() string {
	if s.startTime == nil {
		return "0ms"
	}

	var d time.Duration
	if s.endTime == nil {
		d = time.Since(*s.startTime)
	} else {
		d = s.endTime.Sub(*s.startTime)
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
func (s *status) Update(new *status) *status {
	if s == nil {
		s = &status{}
	}
	if new.offset > 0 {
		s.descriptor = new.descriptor
		s.offset = new.offset
	}
	if new.prompt != "" {
		s.prompt = new.prompt
	}
	if new.startTime != nil {
		s.startTime = new.startTime
	}
	if new.endTime != nil {
		s.endTime = new.endTime
		s.done = true
	}
	return s
}
