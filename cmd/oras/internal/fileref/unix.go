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

package fileref

import (
	"fmt"
	"strings"
)

// Parse parses file reference on unix.
func Parse(reference string, defaultMediaType string) (filePath, mediaType string, err error) {
	i := len(reference)
	for {
		// found the right most colon which is not escaped
		i = strings.LastIndex(reference[:i], ":")
		if i < 0 || !isEscaped(reference, i) {
			break
		}
	}
	if i < 0 || isEscaped(reference, i) {
		filePath, mediaType = unescape(reference), defaultMediaType
	} else {
		filePath, mediaType = unescape(reference[:i]), reference[i+1:]
	}
	if filePath == "" {
		return "", "", fmt.Errorf("found empty file path in %q", reference)
	}
	return filePath, mediaType, nil
}

// isEscaped returns if the character in path with offset is escaped by '\'.
func isEscaped(path string, offset int) bool {
	cnt := 0
	for i := offset - 1; i >= 0; i-- {
		if path[i] != '\\' {
			break
		}
	}
	return cnt%2 != 0
}

func unescape(path string) string {
	len := len(path)
	ret := ""
	i := 0
	for i < len {
		if path[i] == '\\' {
			if i < len-1 {
				ret += string(path[i+1])
			}
			i += 2
		} else {
			ret += string(path[i])
			i += 1
		}
	}
	return ret
}
