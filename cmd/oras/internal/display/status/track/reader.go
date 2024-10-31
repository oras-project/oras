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
	"oras.land/oras/internal/experimental/track"
)

type reader struct {
	base      io.Reader
	offset    int64
	size      int64
	manager   track.Manager
	messenger track.Tracker
}

// NewReader returns a new reader with tracked progress.
func NewReader(r io.Reader, descriptor ocispec.Descriptor, actionPrompt string, donePrompt string, tty *os.File) (*reader, error) {
	prompt := map[track.State]string{
		track.StateInitialized:  actionPrompt,
		track.StateTransmitting: actionPrompt,
		track.StateTransmitted:  donePrompt,
	}

	manager, err := progress.NewManager(tty, prompt)
	if err != nil {
		return nil, err
	}
	return managedReader(r, descriptor, manager)
}

func managedReader(r io.Reader, descriptor ocispec.Descriptor, manager track.Manager) (*reader, error) {
	messenger, err := manager.Track(descriptor)
	if err != nil {
		return nil, err
	}

	return &reader{
		base:      r,
		size:      descriptor.Size,
		manager:   manager,
		messenger: messenger,
	}, nil
}

// StopManager stops the messenger channel and related manager.
func (r *reader) StopManager() {
	r.Close()
	_ = r.manager.Close()
}

// Done sends message to mark the tracked progress as complete.
func (r *reader) Done() {
	r.messenger.Update(track.Status{
		State:  track.StateTransmitted,
		Offset: r.size,
	})
	r.messenger.Close()
}

// Close closes the update channel.
func (r *reader) Close() {
	r.messenger.Close()
}

// Start sends the start timing to the messenger channel.
func (r *reader) Start() {
	r.messenger.Update(track.Status{
		State:  track.StateInitialized,
		Offset: -1,
	})
}

// Read reads from the underlying reader and updates the progress.
func (r *reader) Read(p []byte) (int, error) {
	n, err := r.base.Read(p)
	if err != nil && err != io.EOF {
		return n, err
	}

	r.offset = r.offset + int64(n)
	if err == io.EOF {
		if r.offset != r.size {
			return n, io.ErrUnexpectedEOF
		}
	}
	r.messenger.Update(track.Status{
		State:  track.StateTransmitting,
		Offset: r.offset,
	})
	return n, err
}
