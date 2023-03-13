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

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

// requestHeaderMatcher provides matching for request content.
// It looks into the debug output of an operation and the match will pass
// if all the headers exist in every sent request.
type requestHeaderMatcher []string

// NewRequestHeaderMatcher returns a request header matcher.
func NewRequestHeaderMatcher(kw []string) requestHeaderMatcher {
	return requestHeaderMatcher(kw)
}

// Match matches got with wanted headers.
func (r requestHeaderMatcher) Match(got *gbytes.Buffer) {
	var missed []string

	raw := string(got.Contents())
	reqs := getRequestHeaders(getRequests(raw))
	for _, req := range reqs {
		for _, w := range r {
			if !strings.Contains(req, w) {
				missed = append(missed, w)
			}
		}
	}

	if len(missed) != 0 {
		fmt.Printf("Headers missed: %v\n", missed)
		panic("failed to match all headers")
	}
}

// getRequests parses raw debug output to a string slice
// containing each request.
func getRequests(debugOutput string) []string {
	reqs := strings.Split(debugOutput, "> Request URL:")
	Expect(len(reqs) > 0).To(BeTrue(), "should output requests in debug logs")
	reqs = reqs[1:]
	// trim the response content
	for i, req := range reqs {
		req = strings.Split(req, "< Response Status:")[0]
		reqs[i] = req
	}
	return reqs
}

// getRequestHeaders takes a string slice containing requests
// and extract request headers from them.
func getRequestHeaders(reqs []string) []string {
	headers := make([]string, len(reqs))
	for i, req := range reqs {
		// extract the header content from each request
		_, header, ok := strings.Cut(req, "> Request headers:\n")
		if ok {
			headers[i] = header
		}
	}
	return headers
}
