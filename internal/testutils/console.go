//go:build !windows

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

package testutils

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/creack/pty"
)

// NewPty creates a new pty pair for testing, caller is responsible for closing
// the returned files if err is not nil.
func NewPty() (*os.File, *os.File, error) {
	return pty.Open()
}

// MatchPty checks that the output matches the expected strings in specified
// order.
func MatchPty(ptmx *os.File, pts *os.File, expected ...string) error {
	var wg sync.WaitGroup
	wg.Add(1)
	var buffer bytes.Buffer
	go func() {
		defer wg.Done()
		_, _ = io.Copy(&buffer, ptmx)
	}()
	_ = pts.Close()
	wg.Wait()

	return OrderedMatch(buffer.String(), expected...)
}

// OrderedMatch matches the got with the expected strings in order.
func OrderedMatch(got string, want ...string) error {
	for _, e := range want {
		i := strings.Index(got, e)
		if i < 0 {
			return fmt.Errorf("failed to find %q in %q", e, got)
		}
		got = got[i+len(e):]
	}
	return nil
}
