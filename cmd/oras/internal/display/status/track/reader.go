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

package track

import (
	"io"
	"os"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/status/progress"
)

type reader struct {
	base         io.Reader
	offset       int64
	actionPrompt string
	donePrompt   string
	descriptor   ocispec.Descriptor
	manager      progress.Manager
	status       progress.Status
}

// NewReader returns a new reader with tracked progress.
func NewReader(r io.Reader, descriptor ocispec.Descriptor, actionPrompt string, donePrompt string, tty *os.File) (*reader, error) {
	manager, err := progress.NewManager(tty)
	if err != nil {
		return nil, err
	}
	return managedReader(r, descriptor, manager, actionPrompt, donePrompt)
}

func managedReader(r io.Reader, descriptor ocispec.Descriptor, manager progress.Manager, actionPrompt string, donePrompt string) (*reader, error) {
	ch, err := manager.Add()
	if err != nil {
		return nil, err
	}

	return &reader{
		base:         r,
		descriptor:   descriptor,
		actionPrompt: actionPrompt,
		donePrompt:   donePrompt,
		manager:      manager,
		status:       ch,
	}, nil
}

// StopManager stops the status channel and related manager.
func (r *reader) StopManager() {
	r.Close()
	_ = r.manager.Close()
}

// Done sends message to mark the tracked progress as complete.
func (r *reader) Done() {
	r.status <- progress.NewStatus(r.donePrompt, r.descriptor, r.descriptor.Size)
	r.status <- progress.EndTiming()
}

// Close closes the update channel.
func (r *reader) Close() {
	close(r.status)
}

// Start sends the start timing to the status channel.
func (r *reader) Start() {
	r.status <- progress.StartTiming()
}

// Read reads from the underlying reader and updates the progress.
func (r *reader) Read(p []byte) (int, error) {
	n, err := r.base.Read(p)
	if err != nil && err != io.EOF {
		return n, err
	}

	r.offset = r.offset + int64(n)
	if err == io.EOF {
		if r.offset != r.descriptor.Size {
			return n, io.ErrUnexpectedEOF
		}
	}
	for {
		select {
		case r.status <- progress.NewStatus(r.actionPrompt, r.descriptor, r.offset):
			// purge the channel until successfully pushed
			return n, err
		case <-r.status:
		}
	}
}
