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
type StateKey struct {
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

// stateMachine specifies options for status log matching.
type stateMachine struct {
	edges map[state][]edge
	start *node
	end   *node
}

func newStateMachine(cmd string) (s *stateMachine) {
	s = &stateMachine{
		start: new(node),
		end:   new(node),
		edges: make(map[string][]edge),
	}

	// prepare edges
	switch cmd {
	case "push", "attach":
		s.addPath("Uploading", "Uploaded")
		s.addPath("Exists")
		s.addPath("Skipped")
	case "pull":
		s.addPath("Downloading", "Downloaded")
		s.addPath("Downloading", "Processing", "Downloaded")
		s.addPath("Skipped")
		s.addPath("Restored")
	default:
		panic("Unrecognized cmd name " + cmd)
	}
	return s
}

func findState(from *node, edges []edge) *edge {
	for _, e := range edges {
		if e.from == from {
			return &e
		}
	}
	return nil
}

func (opts *stateMachine) addPath(s ...string) {
	last := opts.start
	len := len(s)
	for i, name := range s {
		//
		e := edge{from: last}
		if i == len-1 {
			e.to = opts.end
		} else {
			e.to = new(node)
		}
		opts.edges[name] = append(opts.edges[name], e)
	}
}

// status type helps matching status log of a oras command.
type status struct {
	states       map[StateKey]*node
	matchResult  map[state][]StateKey
	successCount int
	verbose      bool

	*stateMachine
}

// NewStatus generates a instance for matchable status logs.
func NewStatus(keys []StateKey, cmd string, verbose bool, successCount int) *status {
	s := status{
		states:       make(map[StateKey]*node),
		matchResult:  make(map[string][]StateKey),
		stateMachine: newStateMachine(cmd),
		successCount: successCount,
		verbose:      verbose,
	}
	for _, k := range keys {
		s.states[k] = s.start
	}
	return &s
}

// switchState moves a node forward in the state machine graph.
func (s *status) switchState(st state, key StateKey) {
	// load state
	now, ok := s.states[key]
	gomega.Expect(ok).To(gomega.BeTrue(), fmt.Sprintf("Should find state node for %v", key))

	// find next
	e := findState(now, s.edges[st])
	gomega.Expect(e).NotTo(gomega.BeNil(), fmt.Sprintf("Should state node not matching for %v, %v", st, key))

	// switch
	s.states[key] = e.to
	if e.to == s.end {
		// collect last state for matching
		s.matchResult[st] = append(s.matchResult[st], key)
	}
}

func (s *status) Match(got []byte) {
	lines := strings.Split(string(got), "\n")
	for _, line := range lines {
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
		key := StateKey{fields[1], fields[2]}
		if _, ok := s.states[key]; !ok {
			// ignore other logs
			continue
		}

		s.switchState(fields[0], key)
	}

	successCnt := 0
	for _, v := range s.matchResult {
		successCnt += len(v)
	}
	gomega.Expect(successCnt).To(gomega.Equal(s.successCount))
}
