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
	"errors"
	"io"
)

// ReadLine reads a line from the reader with trailing \r dropped.
func ReadLine(reader io.Reader) ([]byte, error) {
	var line []byte
	var buffer [1]byte
	drop := 0
	for {
		n, err := reader.Read(buffer[:])
		if err != nil {
			if err == io.EOF {
				// a line ends if reader is closed
				// drop \r if it is the last character
				return line[:len(line)-drop], nil
			}
			return nil, err
		}
		if n != 1 {
			return nil, errors.New("failed to read with 1-byte buffer")
		}
		switch c := buffer[0]; c {
		case '\r':
			drop = 1
			line = append(line, c)
		case '\n':
			// a line ends with \n
			return line[:len(line)-drop], nil
		default:
			drop = 0
			line = append(line, c)
		}
	}
}
