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
	"sync"
	"sync/atomic"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/progress"
)

type reader struct {
	base         io.Reader
	offset       atomic.Uint64
	actionPrompt string
	donePrompt   string
	descriptor   ocispec.Descriptor
	mu           sync.Mutex
	m            progress.Manager
	status       progress.Status
	once         sync.Once
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
		m:            manager,
		status:       ch,
	}, nil
}

// End closes the status channel.
func (r *reader) End() {
	defer close(r.status)
	r.status <- progress.NewStatus(r.donePrompt, r.descriptor, uint64(r.descriptor.Size))
	r.status <- progress.EndTiming()
}

// Stop stops the status channel and related manager.
func (r *reader) Stop() error {
	r.End()
	return r.m.Close()
}

func (r *reader) Read(p []byte) (int, error) {
	r.once.Do(func() {
		r.status <- progress.StartTiming()
	})
	n, err := r.base.Read(p)
	if err != nil && err != io.EOF {
		return n, err
	}

	offset := r.offset.Add(uint64(n))
	if err == io.EOF {
		if offset != uint64(r.descriptor.Size) {
			return n, io.ErrUnexpectedEOF
		}
		r.status <- progress.NewStatus(r.actionPrompt, r.descriptor, offset)
	}

	if len(r.status) < progress.BufferSize {
		// intermediate progress might be ignored if buffer is full
		r.status <- progress.NewStatus(r.actionPrompt, r.descriptor, offset)
	}
	return n, err
}
