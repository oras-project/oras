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

package option

import (
	"errors"
	"testing"
)

func TestFormat_WriteTo_marshalFailure(t *testing.T) {
	opts := Format{Template: "json"}
	err := opts.WriteTo(nil, make(chan int))
	if err == nil {
		t.Errorf("expected json marshal error")
	}
}

type invalidWriter struct{}

func (w *invalidWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("failed")
}
func TestFormat_WriteTo_writeFailure(t *testing.T) {
	opts := Format{Template: "json"}
	err := opts.WriteTo(&invalidWriter{}, nil)
	if err == nil {
		t.Errorf("expected json marshal error")
	}
}
