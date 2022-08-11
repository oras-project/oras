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

package output

import "strings"

type keywordMatcher struct {
	matched int
	visited map[string]bool
}

func NewKeywordMatcher(keywords []string) *keywordMatcher {
	km := keywordMatcher{}
	km.visited = make(map[string]bool)
	for _, w := range keywords {
		km.visited[w] = false
	}
	km.matched = 0
	return &km
}

func (km *keywordMatcher) Write(p []byte) (n int, err error) {
	str := string(p)
	for k, v := range km.visited {
		if !v && strings.Contains(str, k) {
			km.visited[k] = true
			km.matched++
		}
	}
	return len(p), nil
}

func (km *keywordMatcher) Status() (expected int, matched int) {
	return len(km.visited), km.matched
}
