package errors

import "errors"

var (
	ErrInvalidReference = errors.New("image reference format is invalid. Please specify <name:tag|name@digest>")
)
