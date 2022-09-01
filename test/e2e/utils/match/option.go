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

type Option struct {
	Stdout     *entry
	Stderr     *entry
	Stdin      io.Reader
	ShouldFail bool
}

// NewOption generates a result type and returns the pointer.
func NewOption(stdin io.Reader, stdout Matchable, stderr Matchable, shouldFail bool) *Option {
	return &Option{
		Stdout:     entryFromMatchable(stdout),
		Stderr:     entryFromMatchable(stderr),
		Stdin:      stdin,
		ShouldFail: shouldFail,
	}
}

func entryFromMatchable(m Matchable) *entry {
	if m == nil {
		return &entry{io.Discard, nil}
	}
	return &entry{newOutput(), m}
}

// Match matches captured with wanted.
func (o *Option) Match() {
	o.Stdout.match()
	o.Stderr.match()
}
