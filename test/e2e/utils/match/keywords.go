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

// Keywords provides selective matching of the output.
// The match will pass if all key words exists in the output.
type Keywords []string

func (kw Keywords) match(w *output) {
	visited := make(map[string]bool)
	for _, w := range kw {
		visited[w] = false
	}

	str := string(w.readAll())
	for k := range visited {
		if strings.Contains(str, k) {
			delete(visited, k)
		}
	}

	if len(visited) != 0 {
		var missed []string
		for k := range visited {
			missed = append(missed, fmt.Sprintf("%q", k))
		}
		Expect(fmt.Sprintf("Keywords missed: %v ===> ", missed) + fmt.Sprintf("Quoted output: %q", str)).To(Equal(""))
	}
}
