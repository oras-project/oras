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
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/status/progress/humanize"
	"time"
)

// Messenger is progress message channel.
type Messenger struct {
	ch     chan *status
	closed bool
}

// Start initializes the messenger.
func (sm *Messenger) Start() {
	if sm.ch == nil {
		return
	}
	sm.ch <- startTiming()
}

// Send a status message for the specified descriptor.
func (sm *Messenger) Send(prompt string, descriptor ocispec.Descriptor, offset int64) {
	for {
		select {
		case sm.ch <- newStatusMessage(prompt, descriptor, offset):
			return
		case <-sm.ch:
			// purge the channel until successfully pushed
		default:
			// ch is nil
			return
		}
	}
}

// Stop the messenger after sending a end message.
func (sm *Messenger) Stop() {
	if sm.closed {
		return
	}
	sm.ch <- endTiming()
	close(sm.ch)
	sm.closed = true
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
