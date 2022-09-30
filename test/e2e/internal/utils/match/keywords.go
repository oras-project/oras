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
)

// KeywordMatcher provides selective matching of the output.
// The match will pass if all key words existed case-insensitively in the
// output.
type KeywordMatcher []string

func NewKeywordMatcher(kw []string) KeywordMatcher {
	return KeywordMatcher(kw)
}

func (want KeywordMatcher) Match(raw []byte) {
	visited := make(map[string]bool)
	for _, w := range want {
		visited[strings.ToLower(w)] = false
	}

	got := strings.ToLower(string(raw))
	for key := range visited {
		if strings.Contains(got, key) {
			delete(visited, key)
		}
	}

	if len(visited) != 0 {
		var missed []string
		for k := range visited {
			missed = append(missed, fmt.Sprintf("%q", k))
		}
		Expect(fmt.Sprintf("Keywords missed: %v ===> ", missed) + fmt.Sprintf("Quoted output: %q", got)).To(Equal(""))
	}
}
