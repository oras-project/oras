package track

import (
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// State represents the state of a descriptor.
type State int

const (
	StateUnknown State = iota
	StateStarted
	StateStopped
	StateExists
	StateSkipped
	StateMounted
)

// Status represents the status of a descriptor.
type Status struct {
	State  State
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

	// Record records the progress of a descriptor.
	Record(desc ocispec.Descriptor, status Status) error

	// Track starts tracking the progress of a descriptor.
	Track(desc ocispec.Descriptor) (Tracker, error)
}
