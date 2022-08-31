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

import "io"

type Entry struct {
	Writer io.Writer
	m      Matchable
}

func newEntry(m Matchable) Entry {
	return Entry{NewWriter(), m}
}

func (e *Entry) Match() {
	if e.Writer == io.Discard {
		return
	}
	e.m.matchTo(e.Writer)
}
