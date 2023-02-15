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
	"strings"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

// contentMatcher provides whole matching of the output.
type contentMatcher struct {
	s         string
	trimSpace bool
}

// NewContentMatcher returns a content matcher.
func NewContentMatcher(s string, trimSpace bool) contentMatcher {
	if trimSpace {
		s = strings.TrimSpace(s)
	}
	return contentMatcher{
		s:         s,
		trimSpace: trimSpace,
	}
}

// Match matches got with s.
func (c contentMatcher) Match(got *gbytes.Buffer) {
	content := string(got.Contents())
	if c.trimSpace {
		content = strings.TrimSpace(content)
	}
	Expect(content).Should(Equal(c.s))
}
