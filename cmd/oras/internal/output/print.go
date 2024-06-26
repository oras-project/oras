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

package output

import (
	"context"
	"fmt"
	"io"
	"sync"

	"oras.land/oras/internal/descriptor"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
)

// PrintFunc is the function type returned by StatusPrinter.
type PrintFunc func(ocispec.Descriptor) error

// Printer prints for status handlers.
type Printer struct {
	out     io.Writer
	err     io.Writer
	verbose bool
	lock    sync.Mutex
}

// NewPrinter creates a new Printer.
func NewPrinter(out io.Writer, err io.Writer, verbose bool) *Printer {
	return &Printer{out: out, err: err, verbose: verbose}
}

// Write implements the io.Writer interface.
func (p *Printer) Write(b []byte) (int, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.out.Write(b)
}

// Println prints objects concurrent-safely with newline.
func (p *Printer) Println(a ...any) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	_, err := fmt.Fprintln(p.out, a...)
	if err != nil {
		err = fmt.Errorf("display output error: %w", err)
		_, _ = fmt.Fprint(p.err, err)
	}
	// Errors are handled above, so return nil
	return nil
}

// Printf prints objects concurrent-safely with newline.
func (p *Printer) Printf(format string, a ...any) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	_, err := fmt.Fprintf(p.out, format, a...)
	if err != nil {
		err = fmt.Errorf("display output error: %w", err)
		_, _ = fmt.Fprint(p.err, err)
	}
	// Errors are handled above, so return nil
	return nil
}

// PrintVerbose prints when verbose is true.
func (p *Printer) PrintVerbose(a ...any) error {
	if !p.verbose {
		return nil
	}
	return p.Println(a...)
}

// PrintStatus prints transfer status.
func (p *Printer) PrintStatus(desc ocispec.Descriptor, status string) error {
	name, isTitle := descriptor.GetTitleOrMediaType(desc)
	if !isTitle {
		return p.PrintVerbose(status, descriptor.ShortDigest(desc), name)
	}
	return p.Println(status, descriptor.ShortDigest(desc), name)
}

// StatusPrinter returns a tracking function for transfer status.
func (p *Printer) StatusPrinter(status string) PrintFunc {
	return func(desc ocispec.Descriptor) error {
		return p.PrintStatus(desc, status)
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
