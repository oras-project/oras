package errors

import "errors"

func ErrInvalidReference(reference string) error {
	return errors.New("image reference format is invalid. Expected <name:tag|name@digest>, got '" + reference + "'")
}
