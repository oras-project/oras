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
	"github.com/onsi/gomega"
)

// Matching error execution.
var Error = Option{nil, nil, nil, true}

// Matching success execution.
var Success = Option{nil, nil, nil, false}

// ErrorKeywords returns an option for matching stderr keywords in failure
// execution.
func ErrorKeywords(keywords ...string) *Option {
	return NewOption(nil, nil, Keywords(keywords), true)
}

// SuccessKeywords returns an option for matching stdout keywords in success
// execution.
func SuccessKeywords(keywords ...string) *Option {
	return NewOption(nil, Keywords(keywords), nil, false)
}

// SuccessContent returns an option for matching stdout content in success
// execution.
func SuccessContent(content string) *Option {
	return NewOption(nil, Content(content), nil, false)
}

func MatchableStatus(cmd string, verbose bool) (starts []state, opts *StatusOption) {
	// prepare states map
	var sequence, directEnds []state
	switch cmd {
	case "push":
		sequence = []state{"Uploading", "Uploaded"}
		if verbose {
			sequence = append([]state{"Preparing"}, sequence...)
		}
		directEnds = []state{"Skipped", "Exists"}
		starts = append(starts, append(directEnds, sequence[0])...)
	default:
		panic("Unrecognized cmd name " + cmd)
	}
	edges := formStatusOption(sequence, directEnds)

	return starts, opts
}

func formStatusOption(sequence []state, directEnds []state) (transitions map[state]*state) {
	gomega.Expect(len(sequence) > 0).Should(gomega.BeTrue())

	transitions = make(map[state]*state)
	last = state(sequence[0])
	for i := 1; i < len(sequence); i++ {
		curr := state(sequence[i])
		transitions[last] = &curr
		last = curr
	}
	transitions[last] = &EndState

	for _, d := range directEnds {
		transitions[state(d)] = &EndState
	}
	return
}
