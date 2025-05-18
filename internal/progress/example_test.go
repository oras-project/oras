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

package progress_test

import (
	"crypto/rand"
	"fmt"
	"io"

	"oras.land/oras/internal/progress"
)

// ExampleTrackReader demonstrates how to track the transmission progress of a
// reader.
func ExampleTrackReader() {
	// Set up a progress tracker.
	total := int64(11)
	tracker := progress.TrackerFunc(func(status progress.Status, err error) error {
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return nil
		}
		switch status.State {
		case progress.StateInitialized:
			fmt.Println("Start reading content")
		case progress.StateTransmitting:
			fmt.Printf("Progress: %d/%d bytes\n", status.Offset, total)
		case progress.StateTransmitted:
			fmt.Println("Finish reading content")
		default:
			// Ignore other states.
		}
		return nil
	})
	// Close takes no effect for TrackerFunc but should be called for general
	// Tracker implementations.
	defer func() { _ = tracker.Close() }()

	// Wrap a reader of a random content generator with the progress tracker.
	r := io.LimitReader(rand.Reader, total)
	rc := progress.TrackReader(tracker, r)

	// Start tracking the transmission.
	if err := progress.Start(tracker); err != nil {
		panic(err)
	}

	// Read from the random content generator and discard the content, while
	// tracking the progress.
	// Note: io.Discard is wrapped with a io.MultiWriter for dropping
	// the io.ReadFrom interface for demonstration purposes.
	buf := make([]byte, 3)
	w := io.MultiWriter(io.Discard)
	if _, err := io.CopyBuffer(w, rc, buf); err != nil {
		panic(err)
	}

	// Finish tracking the transmission.
	if err := progress.Done(tracker); err != nil {
		panic(err)
	}

	// Output:
	// Start reading content
	// Progress: 3/11 bytes
	// Progress: 6/11 bytes
	// Progress: 9/11 bytes
	// Progress: 11/11 bytes
	// Finish reading content
}
