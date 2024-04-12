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
	"os"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras/internal/listener"
)

// Types and functions in this file are deprecated and should be removed when
// no-longer referenced.

// NewTagStatusHintPrinter creates a wrapper type for printing
// tag status and hint.
func NewTagStatusHintPrinter(target oras.Target, refPrefix string) oras.Target {
	var printHint sync.Once
	var printHintErr error
	onTagging := func(desc ocispec.Descriptor, tag string) error {
		printHint.Do(func() {
			ref := refPrefix + "@" + desc.Digest.String()
			printHintErr = Print("Tagging", ref)
		})
		return printHintErr
	}
	onTagged := func(desc ocispec.Descriptor, tag string) error {
		return Print("Tagged", tag)
	}
	return listener.NewTagListener(target, onTagging, onTagged)
}

// NewTagStatusPrinter creates a wrapper type for printing tag status.
func NewTagStatusPrinter(target oras.Target) oras.Target {
	return listener.NewTagListener(target, nil, func(desc ocispec.Descriptor, tag string) error {
		return Print("Tagged", tag)
	})
}

// printer is used by the code being deprecated. Related functions should be
// removed when no-longer referenced.
var printer = NewPrinter(os.Stdout)

// Print objects to display concurrent-safely.
func Print(a ...any) error {
	return printer.Println(a...)
}

// StatusPrinter returns a tracking function for transfer status.
func StatusPrinter(status string, verbose bool) PrintFunc {
	return printer.StatusPrinter(status, verbose)
}

// PrintStatus prints transfer status.
func PrintStatus(desc ocispec.Descriptor, status string, verbose bool) error {
	return printer.PrintStatus(desc, status, verbose)
}
