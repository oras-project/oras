package oras

import "errors"

// Common errors
var (
	ErrNotFound          = errors.New("not_found")
	ErrResolverUndefined = errors.New("resolver_undefined")
	ErrEmptyContents     = errors.New("empty_contents")
)
