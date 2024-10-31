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
	"time"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/status/progress/humanize"
	"oras.land/oras/internal/experimental/track"
)

// Messenger is progress message channel.
type Messenger struct {
	ch     chan *status
	closed bool
	desc   ocispec.Descriptor
	prompt map[track.State]string
}

func (m *Messenger) Update(status track.Status) error {
	if status.State == track.StateInitialized {
		m.start()
	}
	m.send(m.prompt[status.State], status.Offset)
	return nil
}

func (m *Messenger) Fail(err error) error {
	return err
}

func (m *Messenger) Close() error {
	m.stop()
	return nil
}

// start initializes the messenger.
func (m *Messenger) start() {
	if m.ch == nil {
		return
	}
	m.ch <- startTiming()
}

// send a status message for the specified descriptor.
func (m *Messenger) send(prompt string, offset int64) {
	for {
		select {
		case m.ch <- newStatusMessage(prompt, m.desc, offset):
			return
		case <-m.ch:
			// purge the channel until successfully pushed
		default:
			// ch is nil
			return
		}
	}
}

// stop the messenger after sending a end message.
func (m *Messenger) stop() {
	if m.closed {
		return
	}
	m.ch <- endTiming()
	close(m.ch)
	m.closed = true
}

// newStatus generates a base empty status.
func newStatus() *status {
	return &status{
		offset:      -1,
		total:       humanize.ToBytes(0),
		speedWindow: newSpeedWindow(framePerSecond),
	}
}

// newStatusMessage generates a status for messaging.
func newStatusMessage(prompt string, descriptor ocispec.Descriptor, offset int64) *status {
	return &status{
		prompt:     prompt,
		descriptor: descriptor,
		offset:     offset,
	}
}

// startTiming creates start timing message.
func startTiming() *status {
	return &status{
		offset:    -1,
		startTime: time.Now(),
	}
}

// endTiming creates end timing message.
func endTiming() *status {
	return &status{
		offset:  -1,
		endTime: time.Now(),
	}
}
