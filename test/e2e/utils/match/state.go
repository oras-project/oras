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

// state represents the expected value of first field in the status log.
type state = string

// state represents the expected value of second and third fields next status log.
type stateKey struct {
	Digest string
	Name   string
}

type node struct {
	uint // padding to make address unique
}

type edge = struct {
	from *node
	to   *node
}

// statusOption specifies options for status log matching.
//   - trasitions: edges in the state machine graph.
//   - keys: array of identifiers for state subject.
type statusOption struct {
	edges   map[state]edge
	start   *node
	end     *node
	verbose bool
}

func (opts *statusOption) addPath(s ...string) {
	last := opts.start
	for i := 0; i < len(s)-1; i++ {
		e, ok := opts.edges[s[i]]
		if !ok {
			// add to graph
			e = edge{last, new(node)}
			opts.edges[s[i]] = e
		}
		last = e.to
	}
}

// Status type helps matching status log of a oras command.
type Status struct {
	states      map[stateKey]*node
	needMatch   map[stateKey]bool
	matchResult map[state][]stateKey

	statusOption
}

// NewStatus generates a instance for matchable status logs.
func NewStatus(keys []stateKey, verbose bool, opts statusOption) *Status {
	s := Status{
		states:       make(map[stateKey]*node),
		matchResult:  make(map[string][]stateKey),
		statusOption: opts,
	}
	for _, k := range keys {
		// optimize keys query
		s.needMatch[k] = true
	}
	s.verbose = verbose
	s.start = opts.start
	s.end = opts.end
	return &s
}

// switchState moves a node forward in the state machine graph.
func (s *Status) switchState(st state, key stateKey) {
	curr, ok := s.states[key]
	gomega.Expect(ok).To(gomega.BeTrue(), fmt.Sprintf("Should find state node for %v", key))

	e, ok := s.edges[st]
	gomega.Expect(ok).To(gomega.BeTrue(), fmt.Sprintf("Should find state node for %v", st))
	gomega.Expect(e.from).To(gomega.Equal(curr), fmt.Sprintf("Should state node not matching for %v, %v", st, key))

	s.states[key] = e.to
	if e.to == s.end {
		s.matchResult[st] = append(s.matchResult[st], key)
	}
}

func (s *Status) match(w *output) {
	for _, line := range w.readAll() {
		// get state key
		fields := strings.Fields(string(line))

		cnt := len(fields)
		if cnt == 2 && !s.verbose {
			// media type is hidden, add it
			fields = append(fields, "")
		}
		if cnt <= 2 || cnt > 3 {
			continue
		}
		key := stateKey{fields[1], fields[2]}
		if !s.needMatch[key] {
			return
		}

		s.switchState(fields[0], key)
	}
}
