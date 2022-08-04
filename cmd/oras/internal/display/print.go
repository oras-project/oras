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
	"oras.land/oras-go/v2/content"
)

var printLock sync.Mutex

// Print objects to display concurrent-safely
func Print(a ...any) error {
	printLock.Lock()
	defer printLock.Unlock()
	_, err := fmt.Println(a...)
	return err
}

// StatusPrinter returns a tracking function for transfer status.
func StatusPrinter(status string, verbose bool) func(context.Context, ocispec.Descriptor) error {
	return func(ctx context.Context, desc ocispec.Descriptor) error {
		name, ok := desc.Annotations[ocispec.AnnotationTitle]
		if !ok {
			if !verbose {
				return nil
			}
			name = desc.MediaType
		}
		return Print(status, ShortDigest(desc), name)
	}
}

// SuccessorPrinter returns a tracking function to print status for successors.
func SuccessorPrinter(status string, fetcher content.Fetcher, committed map[string]string, verbose bool) func(context.Context, ocispec.Descriptor) error {
	return func(ctx context.Context, desc ocispec.Descriptor) error {
		committed[desc.Digest.String()] = desc.Annotations[ocispec.AnnotationTitle]
		successors, err := content.Successors(ctx, fetcher, desc)
		if err != nil {
			return err
		}
		for _, s := range successors {
			name := s.Annotations[ocispec.AnnotationTitle]
			if committed[s.Digest.String()] != name {
				// Reprint status for deduplicated content
				if err := StatusPrinter(status, verbose)(ctx, s); err != nil {
					return err
				}
			}
		}
		return nil
	}
}
