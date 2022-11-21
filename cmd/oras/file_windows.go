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

package main

import (
	"strings"
	"unicode"
)

// parseFileReference parses file reference on windows.
// Windows systems does not allow ':' in the file path except for drive letter.
func parseFileReference(reference string, mediaType string) (filePath, mediatype string) {
	i := strings.LastIndex(reference, ":")
	if i < 0 {
		return reference, mediaType
	}

	// In case it is C:\
	if i == 1 && len(reference) > 2 && reference[2] == '\\' && unicode.IsLetter(rune(reference[0])) {
		return reference, mediaType
	}
	return reference[:i], reference[i+1:]
}
