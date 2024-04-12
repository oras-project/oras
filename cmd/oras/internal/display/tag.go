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
	"io"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry"
)

// NewTagStatusHintPrinter creates a wrapper type for printing
// tag status and hint.
func NewTagStatusHintPrinter(target oras.Target, preTagging func(desc ocispec.Descriptor) error, onTaggedStatus func(tag string) error, onTaggedMetadata func(tag string) error) oras.Target {
	onTagged := func(tag string) error {
		if err := onTaggedStatus(tag); err != nil {
			return err
		}
		return onTaggedMetadata(tag)
	}
	if repo, ok := target.(registry.Repository); ok {
		return &tagManifestStatusForRepo{
			Repository: repo,
			preTagging: preTagging,
			onTagged:   onTagged,
		}
	}
	return &tagManifestStatusForTarget{
		Target:     target,
		preTagging: preTagging,
		onTagged:   onTagged,
	}
}

type tagManifestStatusForRepo struct {
	printHint sync.Once
	registry.Repository
	onTagged   func(tag string) error
	preTagging func(desc ocispec.Descriptor) error
}

// PushReference overrides Repository.PushReference method to print off which tag(s) were added successfully.
func (p *tagManifestStatusForRepo) PushReference(ctx context.Context, expected ocispec.Descriptor, content io.Reader, reference string) error {
	var err error
	p.printHint.Do(func() {
		err = p.preTagging(expected)
	})
	if err != nil {
		return err
	}
	if err := p.Repository.PushReference(ctx, expected, content, reference); err != nil {
		return err
	}
	return p.onTagged(reference)
}

type tagManifestStatusForTarget struct {
	printHint sync.Once
	oras.Target
	onTagged   func(tag string) error
	preTagging func(desc ocispec.Descriptor) error
}

// Tag tags a descriptor with a reference string.
func (p *tagManifestStatusForTarget) Tag(ctx context.Context, desc ocispec.Descriptor, reference string) error {
	var err error
	p.printHint.Do(func() {
		err = p.preTagging(desc)
	})
	if err != nil {
		return err
	}
	if err := p.Target.Tag(ctx, desc, reference); err != nil {
		return err
	}
	return p.onTagged(reference)
}
