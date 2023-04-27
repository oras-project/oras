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
	"github.com/onsi/gomega/gbytes"
)

// keywordMatcher provides selective matching of the output.
// The match will pass if all the keywords exist case-insensitively
// in the output.
type keywordMatcher []string

func NewKeywordMatcher(kw []string) keywordMatcher {
	return keywordMatcher(kw)
}

// Match matches got with wanted keywords.
func (want keywordMatcher) Match(got *gbytes.Buffer) {
	var missed []string

	raw := string(got.Contents())
	lowered := strings.ToLower(raw)
	for _, w := range want {
		if !strings.Contains(lowered, strings.ToLower(w)) {
			missed = append(missed, w)
		}
	}

	if len(missed) != 0 {
		fmt.Printf("Keywords missed: %v\n", missed)
		ginkgo.Fail("failed to match all keywords")
	}
}
