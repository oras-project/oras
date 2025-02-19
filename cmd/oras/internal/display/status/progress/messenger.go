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

// messenger is progress message channel.
type messenger struct {
	update  chan statusUpdate
	closed  bool
	prompts map[progress.State]string
}

// Update sends the status to the message channel.
func (m *messenger) Update(status progress.Status) error {
	switch status.State {
	case progress.StateInitialized:
		m.update <- updateStatusStartTime()
	case progress.StateTransmitting:
		select {
		case m.update <- updateStatusMessage(m.prompts[progress.StateTransmitting], status.Offset):
		default:
			// drop message if channel is full
		}
	default:
		m.update <- updateStatusMessage(m.prompts[status.State], status.Offset)
	}
	return nil
}

// Fail sends the error to the message channel.
func (m *messenger) Fail(err error) error {
	m.update <- updateStatusError(err)
	return nil
}

// Close marks the progress as completed and closes the message channel.
func (m *messenger) Close() error {
	if m.closed {
		return nil
	}
	m.update <- updateStatusEndTime()
	close(m.update)
	m.closed = true
	return nil
}
