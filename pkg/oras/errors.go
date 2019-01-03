package oras

import "errors"

// Common errors
var (
	ErrResolverUndefined = errors.New("resolver_undefined")
	ErrEmptyDescriptors  = errors.New("empty_descriptors")
)
