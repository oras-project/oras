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
	"os"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry"
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
	return p.Println(status, ShortDigest(desc), name)
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

// NewTagStatusHintPrinter creates a wrapper type for printing
// tag status and hint.
func NewTagStatusHintPrinter(target oras.Target, refPrefix string) oras.Target {
	var printHint sync.Once
	if repo, ok := target.(registry.Repository); ok {
		return &tagManifestStatusForRepo{
			Repository: repo,
			printHint:  &printHint,
			refPrefix:  refPrefix,
		}
	}
	return &tagManifestStatusForTarget{
		Target:    target,
		printHint: &printHint,
		refPrefix: refPrefix,
	}
}

type tagManifestStatusForRepo struct {
	registry.Repository
	printHint *sync.Once
	refPrefix string
}

// PushReference overrides Repository.PushReference method to print off which tag(s) were added successfully.
func (p *tagManifestStatusForRepo) PushReference(ctx context.Context, expected ocispec.Descriptor, content io.Reader, reference string) error {
	if p.printHint != nil {
		p.printHint.Do(func() {
			ref := p.refPrefix + "@" + expected.Digest.String()
			_ = Print("Tagging", ref)
		})
	}
	if err := p.Repository.PushReference(ctx, expected, content, reference); err != nil {
		return err
	}
	return Print("Tagged", reference)
}

type tagManifestStatusForTarget struct {
	oras.Target
	printHint *sync.Once
	refPrefix string
}

// Tag tags a descriptor with a reference string.
func (p *tagManifestStatusForTarget) Tag(ctx context.Context, desc ocispec.Descriptor, reference string) error {
	if p.printHint != nil {
		p.printHint.Do(func() {
			ref := p.refPrefix + "@" + desc.Digest.String()
			_ = Print("Tagging", ref)
		})
	}

	if err := p.Target.Tag(ctx, desc, reference); err != nil {
		return err
	}
	return Print("Tagged", reference)
}

// NewTagStatusPrinter creates a wrapper type for printing tag status.
func NewTagStatusPrinter(target oras.Target) oras.Target {
	if repo, ok := target.(registry.Repository); ok {
		return &tagManifestStatusForRepo{
			Repository: repo,
		}
	}
	return &tagManifestStatusForTarget{
		Target: target,
	}
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
