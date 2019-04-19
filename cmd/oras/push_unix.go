// +build !windows

package main

import "strings"

func parseFileRef(ref string, mediaType string) (string, string) {
	i := strings.LastIndex(ref, ":")
	if i < 0 {
		return ref, mediaType
	}
	return ref[:i], ref[i+1:]
}
