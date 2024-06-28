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

package status

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras/cmd/oras/internal/output"
	"oras.land/oras/internal/listener"
)

// Types and functions in this file are deprecated and should be removed when
// no-longer referenced.

// NewTagStatusPrinter creates a wrapper type for printing tag status.
func NewTagStatusPrinter(printer *output.Printer, target oras.Target) oras.Target {
	return listener.NewTagListener(target, nil, func(desc ocispec.Descriptor, tag string) error {
		return printer.Println("Tagged", tag)
	})
}
