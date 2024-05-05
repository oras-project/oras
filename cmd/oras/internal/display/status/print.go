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
	"context"
	"fmt"
	"io"
	"oras.land/oras/internal/descriptor"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
)

// PrintFunc is the function type returned by StatusPrinter.
type PrintFunc func(ocispec.Descriptor) error

// Printer prints for status handlers.
type Printer struct {
	out  io.Writer
	lock sync.Mutex
}

// NewPrinter creates a new Printer.
func NewPrinter(out io.Writer) *Printer {
	return &Printer{out: out}
}

// Println prints objects concurrent-safely with newline.
func (p *Printer) Println(a ...any) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	_, err := fmt.Fprintln(p.out, a...)
	return err
}

// PrintStatus prints transfer status.
func (p *Printer) PrintStatus(desc ocispec.Descriptor, status string, verbose bool) error {
	name, ok := desc.Annotations[ocispec.AnnotationTitle]
	if !ok {
		// no status for unnamed content
		if !verbose {
			return nil
		}
		name = desc.MediaType
	}
	return p.Println(status, descriptor.ShortDigest(desc), name)
}

// StatusPrinter returns a tracking function for transfer status.
func (p *Printer) StatusPrinter(status string, verbose bool) PrintFunc {
	return func(desc ocispec.Descriptor) error {
		return p.PrintStatus(desc, status, verbose)
	}
}

// PrintSuccessorStatus prints transfer status of successors.
func PrintSuccessorStatus(ctx context.Context, desc ocispec.Descriptor, fetcher content.Fetcher, committed *sync.Map, print PrintFunc) error {
	successors, err := content.Successors(ctx, fetcher, desc)
	if err != nil {
		return err
	}
	for _, s := range successors {
		name := s.Annotations[ocispec.AnnotationTitle]
		if v, ok := committed.Load(s.Digest.String()); ok && v != name {
			// Reprint status for deduplicated content
			if err := print(s); err != nil {
				return err
			}
		}
	}
	return nil
}
