//go:build !windows
// +build !windows

package input

import "strings"

// ParseFileReference parse file reference on windows.
func ParseFileReference(reference string, mediaType string) (filePath, mediatype string) {

	i := strings.LastIndex(reference, ":")
	if i < 0 {
		return indator, mediaType
	}
	return indator[:i], indator[i+1:]
}
