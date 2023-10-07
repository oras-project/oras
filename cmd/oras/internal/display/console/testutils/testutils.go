//go:build darwin || freebsd || linux || netbsd || openbsd || solaris

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

	"github.com/containerd/console"
)

// NewPty creates a new pty pair for testing, caller is responsible for closing
// the returned slave if err is not nil.
func NewPty() (console.Console, *os.File, error) {
	pty, slavePath, err := console.NewPty()
	if err != nil {
		return nil, nil, err
	}
	slave, err := os.OpenFile(slavePath, os.O_RDWR, 0)
	if err != nil {
		return nil, nil, err
	}
	return pty, slave, nil
}

// OrderedMatch checks that the output from the pty matches the expected strings
func OrderedMatch(pty console.Console, slave *os.File, expected ...string) error {
	var wg sync.WaitGroup
	wg.Add(1)
	var buffer bytes.Buffer
	go func() {
		defer wg.Done()
		_, _ = io.Copy(&buffer, pty)
	}()
	slave.Close()
	wg.Wait()

	got := buffer.String()
	for _, e := range expected {
		i := strings.Index(got, e)
		if i < 0 {
			return fmt.Errorf("failed to find %q in %q", e, got)
		}
		got = got[i+len(e):]
	}
	return nil
}
