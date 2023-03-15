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

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

// requestHeaderMatcher provides matching for request headers.
// Given a url prefix, it looks at the requests sent to the urls which
// match the given prefix, and check if these requests all contain
// the specified headers.
type requestHeaderMatcher struct {
	urlPrefix string
	headers   []string
}

// MatchCpRequestHeaders returns a request header matcher
// with the given url prefix.
func NewRequestHeaderMatcher(urlPrefix string, headers []string) requestHeaderMatcher {
	return requestHeaderMatcher{urlPrefix, headers}
}

// Match matches got with wanted headers.
func (r requestHeaderMatcher) Match(got *gbytes.Buffer) {
	var missed []string

	raw := string(got.Contents())
	reqs := getRequestHeaders(getRequests(r.urlPrefix, raw))
	for _, req := range reqs {
		for _, h := range r.headers {
			if !strings.Contains(req, h) {
				missed = append(missed, h)
			}
		}
	}

	if len(missed) != 0 {
		fmt.Printf("Headers missed: %v\n", missed)
		ginkgo.Fail("failed to match all headers")
	}
}

// getRequests parses raw debug output to a string slice
// containing each request that match the given prefix.
func getRequests(urlPrefix string, debugOutput string) []string {
	reqs := strings.Split(debugOutput, "Request #")
	Expect(len(reqs) > 0).To(BeTrue(), "should output requests in debug logs")
	reqs = reqs[1:]
	// trim the response content
	for i, req := range reqs {
		req = strings.Split(req, "Response #")[0]
		reqs[i] = req
	}
	if urlPrefix == "" {
		return reqs
	}
	// filter with the url prefix
	filteredReqs := []string{}
	for _, req := range reqs {
		// extract request url to match the prefix
		_, rest, ok := strings.Cut(req, urlPrefix)
		if ok {
			filteredReqs = append(filteredReqs, rest)
		}
	}
	return filteredReqs
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
