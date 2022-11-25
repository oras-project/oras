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

package file

import (
	"path/filepath"
	"strings"
	"unicode"
)

func doParse(reference string, mediaType string) (filePath, mediatype string) {
	i := strings.LastIndex(reference, ":")
	if i < 0 {
		return reference, mediaType
	}
	// In case it is C:\
	if i == 1 && len(reference) > 2 && unicode.IsLetter(rune(reference[0])) {
		if reference[2] != '\\' {
			if abs, err := filepath.Abs(reference); err == nil {
				return abs, mediaType
			}
		}
		return reference, mediaType
	}
	return reference[:i], reference[i+1:]
}
