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

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// Status writes status output.
type Status struct {
	lock sync.Mutex
	out  io.Writer
}

// NewStatus returns a new status struct for pull operation.
func NewStatus() *Status {
	return &Status{
		out: os.Stdout,
	}
}

// print outputs status with locking.
func (s *Status) Print(a ...any) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	_, err := fmt.Fprintln(s.out, a...)
	return err
}
