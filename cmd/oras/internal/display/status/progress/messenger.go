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

import "oras.land/oras/internal/progress"

// Messenger is progress message channel.
type Messenger struct {
	update chan statusUpdate
	closed bool
	prompt map[progress.State]string
}

func (m *Messenger) Update(status progress.Status) error {
	if status.State == progress.StateInitialized {
		m.start()
	}
	m.send(m.prompt[status.State], status.Offset)
	return nil
}

func (m *Messenger) Fail(err error) error {
	m.update <- updateStatusError(err)
	return nil
}

func (m *Messenger) Close() error {
	m.stop()
	return nil
}

// start initializes the messenger.
func (m *Messenger) start() {
	if m.update == nil {
		return
	}
	m.update <- updateStatusStartTime()
}

// send a status message for the specified descriptor.
func (m *Messenger) send(prompt string, offset int64) {
	for {
		select {
		case m.update <- updateStatusMessage(prompt, offset):
			return
		case <-m.update:
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
	m.update <- updateStatusEndTime()
	close(m.update)
	m.closed = true
}
