package main

import (
	"strings"
	"unicode"
)

// parseFileRef parse file reference on windows.
// Windows systems does not allow ':' in the file path except for drive letter.
func parseFileRef(ref string, mediaType string) (string, string) {
	i := strings.Index(ref, ":")
	if i < 0 {
		return ref, mediaType
	}

	// In case it is C:\
	if i == 1 && len(ref) > 2 && ref[2] == '\\' && unicode.IsLetter(rune(ref[0])) {
		i = strings.Index(ref[3:], ":")
		if i < 0 {
			return ref, mediaType
		}
		i += 3
	}

	return ref[:i], ref[i+1:]
}
