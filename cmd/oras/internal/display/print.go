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

package display

import (
	"context"
	"fmt"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

var printLock sync.Mutex

// Print objects to display concurrent-safely
func Print(a ...any) error {
	printLock.Lock()
	defer printLock.Unlock()
	_, err := fmt.Println(a...)
	return err
}

// NamedStatusPrinter returns a tracking function for transfer status with names.
func StatusPrinter(status string, getNames func(desc ocispec.Descriptor) []string, verbose bool) func(context.Context, ocispec.Descriptor) error {
	return func(ctx context.Context, desc ocispec.Descriptor) error {
		var names []string
		if getNames != nil {
			names = getNames(desc)
		}
		if len(names) == 0 {
			// no name found
			if !verbose {
				return nil
			}
			names = []string{desc.MediaType}
		}
		for _, n := range names {
			if err := Print(status, ShortDigest(desc), n); err != nil {
				return err
			}
		}
		return nil
	}
	return Print(status, ShortDigest(desc), name)
}
