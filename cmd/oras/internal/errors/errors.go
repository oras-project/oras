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

package errors

import (
	"errors"
	"fmt"

	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
)

// NewErrInvalidReference creates a new error based on the reference string.
func NewErrInvalidReference(ref registry.Reference) error {
	return NewErrInvalidReferenceStr(ref.String())
}

// NewErrInvalidReferenceStr creates a new error based on the reference string.
func NewErrInvalidReferenceStr(ref string) error {
	return fmt.Errorf("%s: invalid image reference, expecting <name:tag|name@digest>", ref)
}

// IsReferrersIndexDelete checks if err is a referrers index delete error.
func IsReferrersIndexDelete(err error) bool {
	var re *remote.ReferrersError
	return errors.As(err, &re) && re.IsReferrersIndexDelete()
}
