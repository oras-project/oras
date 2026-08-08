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
	"sync"

	"oras.land/oras/internal/progress"
)

// messenger is progress message channel.
//
// The update channel is deliberately never closed. Trackers are handed to
// io.Reader / io.Writer wrappers that can outlive the operation they track,
// most notably the net/http request body reader, which keeps being read by
// the transport write loop after the round trip returns. Closing the channel
// would let such a late Update or Fail send on a closed channel and panic.
// Instead, Close closes the done channel and every send races the done signal,
// so late updates are dropped instead of panicking.
type messenger struct {
	update    chan statusUpdate
	done      chan struct{}
	closeOnce sync.Once
	prompts   map[progress.State]string
}

// newMessenger creates a new messenger.
func newMessenger(prompts map[progress.State]string) *messenger {
	return &messenger{
		update:  make(chan statusUpdate, bufferSize),
		done:    make(chan struct{}),
		prompts: prompts,
	}
}

// send delivers the update unless the messenger is already closed.
func (m *messenger) send(update statusUpdate) {
	select {
	case m.update <- update:
	case <-m.done:
		// the messenger is closed, drop the update
	}
}

// Update sends the status to the message channel.
func (m *messenger) Update(status progress.Status) error {
	switch status.State {
	case progress.StateInitialized:
		m.send(updateStatusStartTime())
	case progress.StateTransmitting:
		select {
		case m.update <- updateStatusMessage(m.prompts[progress.StateTransmitting], status.Offset):
		default:
			// drop message if channel is full or the messenger is closed
		}
	default:
		m.send(updateStatusMessage(m.prompts[status.State], status.Offset))
	}
	return nil
}

// Fail sends the error to the message channel.
func (m *messenger) Fail(err error) error {
	m.send(updateStatusError(err))
	return nil
}

// Close marks the progress as completed and stops the message channel.
func (m *messenger) Close() error {
	m.closeOnce.Do(func() {
		// the done channel is still open, so this update is guaranteed to be
		// delivered to the drain goroutine before it stops
		m.send(updateStatusEndTime())
		close(m.done)
	})
	return nil
}

// drain applies the status updates to s until the messenger is closed.
func (m *messenger) drain(s *status) {
	for {
		select {
		case update := <-m.update:
			update(s)
		case <-m.done:
			// apply the updates that are still buffered, then stop
			for {
				select {
				case update := <-m.update:
					update(s)
				default:
					return
				}
			}
		}
	}
}
