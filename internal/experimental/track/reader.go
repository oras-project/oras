package track

import "io"

// ReadTracker tracks the transmission based on the read operation.
type ReadTracker struct {
	base    io.Reader
	tracker Tracker
	offset  int64
}

// NewReadTracker attaches a tracker to a reader.
func NewReadTracker(track Tracker, r io.Reader) *ReadTracker {
	return &ReadTracker{
		base:    r,
		tracker: track,
	}
}

// Read reads from the base reader and updates the status.
func (rt *ReadTracker) Read(p []byte) (n int, err error) {
	n, err = rt.base.Read(p)
	rt.offset += int64(n)
	_ = rt.tracker.Update(Status{
		State:  StateTransmitting,
		Offset: rt.offset,
	})
	if err != nil && err != io.EOF {
		_ = rt.tracker.Fail(err)
	}
	return n, err
}

// Close closes the tracker.
func (rt *ReadTracker) Close() error {
	return rt.tracker.Close()
}

// Start starts tracking the transmission.
func (rt *ReadTracker) Start() error {
	return rt.tracker.Update(Status{
		State:  StateInitialized,
		Offset: -1,
	})
}

// Done marks the transmission as complete.
// Done should be called after the transmission is complete.
// Note: Reading all content from the reader does not imply the transmission is
// complete.
func (rt *ReadTracker) Done() error {
	return rt.tracker.Update(Status{
		State:  StateTransmitted,
		Offset: -1,
	})
}
