//go:build windows

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
	"unicode"
)

// Parse parses file reference into filePath and metadata.
func Parse(reference string, defaultMetadata string) (filePath, metadata string, err error) {
	filePath, metadata = doParse(reference, defaultMetadata)
	if filePath == "" {
		return "", "", fmt.Errorf("found empty file path in %q", reference)
	}
	if strings.ContainsAny(filePath, `<>"|?*`) {
		// Reference: https://learn.microsoft.com/windows/win32/fileio/naming-a-file#naming-conventions
		return "", "", fmt.Errorf("reserved characters found in the file path: %s", filePath)
	}
	return filePath, metadata, nil
}

func doParse(reference string, defaultMetadata string) (filePath, metadata string) {
	i := strings.LastIndex(reference, ":")
	if i < 0 || (i == 1 && len(reference) > 2 && unicode.IsLetter(rune(reference[0])) && reference[2] == '\\') {
		// Relative file path with disk prefix is NOT supported, e.g. `c:file1`
		return reference, defaultMetadata
	}
	return reference[:i], reference[i+1:]
}
