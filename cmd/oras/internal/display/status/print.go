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
	"oras.land/oras-go/v2/registry"
)

type Printer struct {
	printLock sync.Mutex
	verbose   bool
	errors    bool
}

var printer Printer

// NewPrinter creates a printer object
func NewPrinter(verbose bool) *Printer {
	printer = Printer{verbose: verbose}
	return &printer
}

// Print objects to display concurrent-safely.
func (p *Printer) Print(a ...any) {
	p.printLock.Lock()
	defer p.printLock.Unlock()
	_, err := fmt.Println(a...)
	if err != nil {
		if !p.errors {
			p.errors = true
			_, _ = fmt.Fprintf(os.Stderr, "Display output error: %w\n", err)
			return
		}
	}
	return
}

// PrintVerbose display in verbose mode.
func (p *Printer) PrintVerbose(a ...any) {
	if p.verbose {
		p.Print(a...)
	}
	return
}

// Print objects to display concurrent-safely.
func Print(a ...any) {
	printer.Print(a...)
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
			Print("Tagging", ref)
		})
	}
	if err := p.Repository.PushReference(ctx, expected, content, reference); err != nil {
		return err
	}
	Print("Tagged", reference)
	return nil
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
			Print("Tagging", ref)
		})
	}

	if err := p.Target.Tag(ctx, desc, reference); err != nil {
		return err
	}
	Print("Tagged", reference)
	return nil
}
