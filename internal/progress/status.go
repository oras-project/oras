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

// State represents the state of a descriptor.
type State int

// Registered states.
const (
	StateUnknown      State = iota // unknown state
	StateInitialized               // progress initialized
	StateTransmitting              // transmitting content
	StateTransmitted               // content transmitted
	StateExists                    // content exists
	StateSkipped                   // content skipped
	StateMounted                   // content mounted
	StateRestored                  // content restored
)

// Status represents the status of a descriptor.
type Status struct {
	// State represents the state of the descriptor.
	State State

	// Offset represents the current offset of the descriptor.
	// Offset is discarded if set to a negative value.
	Offset int64
}
