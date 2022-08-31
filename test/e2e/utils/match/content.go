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
	"io"

	. "github.com/onsi/gomega"
)

type Content string

func (c Content) NewMatchEntry() Entry {
	if c == "" {
		return Entry{io.Discard, nil}
	}
	return newEntry(c)
}

func (c Content) matchTo(w io.Writer) {
	Expect(string(w.(*writer).ReadAll())).To(Equal(string(c)))
}

// Skip can be used when no matching wanted.
var Skip Content = ""
