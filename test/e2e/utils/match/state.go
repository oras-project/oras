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

package match

import (
	"fmt"
	"strings"

	"github.com/onsi/gomega"
)

type stateKey struct {
	Digest string
	Name   string
}

// state represents the expected value of first field in next status log.
type state string

// EndState represents expecting no more logs.
var EndState state = "All status printed"

// StatusOption specifies options for statsu log matching.
//   - trasitions: edges in the state machine graph.
//   - keys: array of identifiers for state subject.
type StatusOption struct {
	trasitions map[state]*state
	Keys       []stateKey
}

// Status type helps matching status log of a oras command.
type Status struct {
	states     map[stateKey]*state
	trasitions map[state]*state
	starts     []state
	needMatch  map[stateKey]bool
	verbose    bool
}

// NewStatus generates a instance for matchable status logs.
func NewStatus(starts []state, verbose bool, opts *StatusOption) *Status {
	s := Status{
		states:     make(map[stateKey]*state),
		trasitions: opts.trasitions,
	}
	for _, k := range opts.Keys {
		// optimize keys
		s.needMatch[k] = true
	}

	s.starts = starts
	s.verbose = verbose
	return &s
}

// switchState moves a state forward in the state machine.
func (s *Status) switchState(key stateKey) *state {
	current, ok := s.states[key]
	if !ok {
		var found = false
		current = &s.start

		gomega.Expect(ok).NotTo(gomega.BeTrue(), fmt.Sprintf("Should find %v in start states", key))
	}

	next := s.trasitions[*current]
	gomega.Expect(next).NotTo(gomega.BeNil(), fmt.Sprintf("Should find next status for %v", *current))
	s.states[key] = next
	return current
}

func (s *Status) match(w *output) {
	for _, line := range w.readAll() {
		// get state key
		fields := strings.Fields(string(line))

		cnt := len(fields)
		if cnt == 2 && !s.verbose {
			// media type is hidden
			fields = append(fields, "")
		}
		if cnt <= 2 || cnt > 3 {
			continue
		}
		key := stateKey{fields[1], fields[2]}
		if !s.needMatch[key] {
			return
		}

		got := fields[0]
		want := s.switchState(key)
		gomega.Expect(got).To(gomega.Equal(string(*want)), fmt.Sprintf("status missing for %v", key))
	}

}
