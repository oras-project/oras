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
	"errors"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/internal/progress"
)

func Test_messenger_Update(t *testing.T) {
	m := &messenger{
		update: make(chan statusUpdate, 1),
		prompts: map[progress.State]string{
			progress.StateInitialized:  "initialized",
			progress.StateTransmitting: "testing",
			progress.StateTransmitted:  "tested",
		},
	}
	defer func() { _ = m.Close() }()
	desc := ocispec.Descriptor{
		MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
		Size:      1234567890,
		Digest:    "sha256:c775e7b757ede630cd0aa1113bd102661ab38829ca52a6422ab782862f268646",
	}
	s := newStatus(desc)

	// test StateInitialized
	if err := m.Update(progress.Status{
		State:  progress.StateInitialized,
		Offset: -1,
	}); err != nil {
		t.Fatalf("messenger.Update() error = %v, wantErr nil", err)
	}
	update := <-m.update
	update(s)
	if s.startTime.IsZero() {
		t.Errorf("messenger.Update(progress.StateInitialized) startTime = zero, want non-zero")
	}
	select {
	case <-m.update:
		t.Errorf("messenger channel is not empty")
	default:
	}

	// test StateTransmitting
	if err := m.Update(progress.Status{
		State:  progress.StateTransmitting,
		Offset: 42,
	}); err != nil {
		t.Fatalf("messenger.Update() error = %v, wantErr nil", err)
	}

	// messages are dropped if channel is full
	if err := m.Update(progress.Status{
		State:  progress.StateTransmitting,
		Offset: 2048,
	}); err != nil {
		t.Fatalf("messenger.Update() error = %v, wantErr nil", err)
	}
	update = <-m.update
	update(s)
	if s.text != "testing" {
		t.Errorf("messenger.Update(progress.StateTransmitting) text = %q, want %q", s.text, "testing")
	}
	if s.offset != 42 {
		t.Errorf("messenger.Update(progress.StateTransmitting) offset = %d, want %d", s.offset, 42)
	}
	select {
	case <-m.update:
		t.Errorf("messenger channel is not empty")
	default:
	}

	// test StateTransmitted
	if err := m.Update(progress.Status{
		State:  progress.StateTransmitted,
		Offset: desc.Size,
	}); err != nil {
		t.Fatalf("messenger.Update() error = %v, wantErr nil", err)
	}
	update = <-m.update
	update(s)
	if s.text != "tested" {
		t.Errorf("messenger.Update(progress.StateTransmitted) text = %q, want %q", s.text, "tested")
	}
	if s.offset != desc.Size {
		t.Errorf("messenger.Update(progress.StateTransmitted) offset = %d, want %d", s.offset, desc.Size)
	}
	select {
	case <-m.update:
		t.Errorf("messenger channel is not empty")
	default:
	}
}

func Test_messenger_Fail(t *testing.T) {
	m := &messenger{
		update: make(chan statusUpdate, 1),
	}
	defer func() { _ = m.Close() }()
	s := new(status)
	errTest := errors.New("test error")

	if err := m.Fail(errTest); err != nil {
		t.Fatalf("messenger.Fail() error = %v, wantErr nil", err)
	}
	update := <-m.update
	update(s)
	if !errors.Is(errTest, s.err) {
		t.Errorf("messenger.Fail() = %v, want %v", s.err, errTest)
	}
}

func Test_messenger_Close(t *testing.T) {
	m := &messenger{
		update: make(chan statusUpdate, 1),
	}
	if err := m.Close(); err != nil {
		t.Fatalf("messenger.Close() error = %v, wantErr nil", err)
	}
	// double close should not panic or return an error
	if err := m.Close(); err != nil {
		t.Fatalf("messenger.Close() error = %v, wantErr nil", err)
	}
}
