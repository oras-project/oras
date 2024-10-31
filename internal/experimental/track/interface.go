package track

import (
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// State represents the state of a descriptor.
type State int

const (
	StateUnknown State = iota
	StateInitialized
	StateTransmitting
	StateTransmitted
	StateExists
	StateSkipped
	StateMounted
)

// Status represents the status of a descriptor.
type Status struct {
	// State represents the state of the descriptor.
	State State

	// Offset represents the current offset of the descriptor.
	// Offset is discarded if set to a negative value.
	Offset int64
}

// Tracker updates the status of a descriptor.
type Tracker interface {
	io.Closer

	// Update updates the status of the descriptor.
	Update(status Status) error

	// Fail marks the descriptor as failed.
	Fail(err error) error
}

// Manager tracks the progress of multiple descriptors.
type Manager interface {
	io.Closer

	// Track starts tracking the progress of a descriptor.
	Track(desc ocispec.Descriptor) (Tracker, error)
}

// Record records the progress of a descriptor.
func Record(m Manager, desc ocispec.Descriptor, status Status) error {
	tracker, err := m.Track(desc)
	if err != nil {
		return err
	}
	err = tracker.Update(status)
	if err != nil {
		return err
	}
	return tracker.Close()
}
