package content

import "errors"

// Common errors
var (
	ErrNotFound        = errors.New("not_found")
	ErrNoName          = errors.New("no_name")
	ErrUnsupportedSize = errors.New("unsupported_size")
)
