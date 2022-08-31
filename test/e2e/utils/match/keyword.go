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
	"io"
	"strings"

	. "github.com/onsi/gomega"
)

type Keyword []string

func (kw Keyword) NewMatchEntry() Entry {
	if kw == nil {
		return Entry{io.Discard, nil}
	}
	return newEntry(kw)
}

func (kw Keyword) matchTo(w io.Writer) {
	visited := make(map[string]bool)
	for _, w := range kw {
		visited[w] = false
	}

	str := string(w.(*writer).ReadAll())
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
