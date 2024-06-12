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
	"oras.land/oras/cmd/oras/internal/output"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras/internal/listener"
)

// Types and functions in this file are deprecated and should be removed when
// no-longer referenced.

// NewTagStatusHintPrinter creates a wrapper type for printing
// tag status and hint.
func NewTagStatusHintPrinter(printer *output.Printer, target oras.Target, refPrefix string) oras.Target {
	var printHint sync.Once
	var printHintErr error
	onTagging := func(desc ocispec.Descriptor, tag string) error {
		printHint.Do(func() {
			ref := refPrefix + "@" + desc.Digest.String()
			printHintErr = printer.Println("Tagging", ref)
		})
		return printHintErr
	}
	onTagged := func(desc ocispec.Descriptor, tag string) error {
		return printer.Println("Tagged", tag)
	}
	return listener.NewTagListener(target, onTagging, onTagged)
}

// NewTagStatusPrinter creates a wrapper type for printing tag status.
func NewTagStatusPrinter(printer *output.Printer, target oras.Target) oras.Target {
	return listener.NewTagListener(target, nil, func(desc ocispec.Descriptor, tag string) error {
		return printer.Println("Tagged", tag)
	})
}
