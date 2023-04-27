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

package repository

import (
	"fmt"
	"strings"

	"oras.land/oras-go/v2/registry"
)

// ParserRepoPath extracts hostname and namespace from rawReference.
func ParseRepoPath(rawReference string) (hostname, namespace string, err error) {
	rawReference = strings.TrimSuffix(rawReference, "/")
	if strings.Contains(rawReference, "/") {
		var ref registry.Reference
		ref, err = registry.ParseReference(rawReference)
		if err != nil {
			return
		}
		if ref.Reference != "" {
			err = fmt.Errorf("tags or digests should not be provided")
			return
		}
		hostname = ref.Registry
		namespace = ref.Repository + "/"
	} else {
		hostname = rawReference
	}
	return
}
