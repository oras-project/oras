package input

import (
	"strings"
	"unicode"
)

// ParseFileReference parse file reference on windows.
// Windows systems does not allow ':' in the file path except for drive letter.
func ParseFileReference(reference string, mediaType string) (filePath, mediatype string) {
	i := strings.Index(reference, ":")
	if i < 0 {
		return reference, mediaType
	}

	// In case it is C:\
	if i == 1 && len(reference) > 2 && reference[2] == '\\' && unicode.IsLetter(rune(reference[0])) {
		i = strings.Index(reference[3:], ":")
		if i < 0 {
			return reference, mediaType
		}
		i += 3
	}
	return reference[:i], reference[i+1:]
}
