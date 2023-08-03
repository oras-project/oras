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

package io

import (
	"bytes"
	"io"
)

// ReadLine reads a line from the reader with trailing \r dropped.
func ReadLine(reader io.Reader) ([]byte, error) {
	var line []byte
	var buffer [1]byte
	for {
		n, err := reader.Read(buffer[:])
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if n == 0 {
			continue
		}
		c := buffer[0]
		if c == '\n' {
			break
		}
		line = append(line, c)
	}
	return bytes.TrimSuffix(line, []byte{'\r'}), nil
}
