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

package track

import (
	"context"
	"errors"
	"io"
	"os"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras/cmd/oras/internal/display/progress"
)

type Trackable interface {
	oras.GraphTarget
	Close() error
	Prompt(ocispec.Descriptor, string, bool) error
}

type graphTarget struct {
	oras.GraphTarget
	manager      progress.Manager
	actionPrompt string
	donePrompt   string
}

type referenceGraphTarget struct {
	*graphTarget
	registry.ReferencePusher
}

// NewTarget creates a new tracked Target.
func NewTarget(t oras.GraphTarget, actionPrompt, donePrompt string, tty *os.File) (Trackable, error) {
	manager, err := progress.NewManager(tty)
	if err != nil {
		return nil, err
	}
	gt := &graphTarget{
		GraphTarget:  t,
		manager:      manager,
		actionPrompt: actionPrompt,
		donePrompt:   donePrompt,
	}

	if refPusher, ok := t.(registry.ReferencePusher); ok {
		return &referenceGraphTarget{
			graphTarget:     gt,
			ReferencePusher: refPusher,
		}, nil
	}
	return gt, nil
}

// Push pushes the content to the Target with tracking.
func (t *graphTarget) Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) error {
	r, err := managedReader(content, expected, t.manager, t.actionPrompt, t.donePrompt)
	if err != nil {
		return err
	}
	defer r.Close()
	r.Start()
	if err := t.GraphTarget.Push(ctx, expected, r); err != nil {
		return err
	}
	r.Done()
	return nil
}

// PushReference pushes the content to the Target with tracking.
func (rgt *referenceGraphTarget) PushReference(ctx context.Context, expected ocispec.Descriptor, content io.Reader, reference string) error {
	r, err := managedReader(content, expected, rgt.manager, rgt.actionPrompt, rgt.donePrompt)
	if err != nil {
		return err
	}
	defer r.Close()
	r.Start()
	err = rgt.ReferencePusher.PushReference(ctx, expected, r, reference)
	if err != nil {
		return err
	}
	r.Done()
	return nil
}

// Predecessors returns the predecessors of the node if supported.
func (t *graphTarget) Predecessors(ctx context.Context, node ocispec.Descriptor) ([]ocispec.Descriptor, error) {
	return t.GraphTarget.Predecessors(ctx, node)
}

// Close closes the tracking manager.
func (t *graphTarget) Close() error {
	return t.manager.Close()
}

// Prompt prompts the user with the provided prompt and descriptor.
// If Target is not set, only prints status.
func (t *graphTarget) Prompt(desc ocispec.Descriptor, prompt string, verbose bool) error {
	if t == nil {
		// this should not happen
		return errors.New("cannot output progress with nil tracked target")
	}
	status, err := t.manager.Add()
	if err != nil {
		return err
	}
	defer close(status)
	status <- progress.NewStatus(prompt, desc, desc.Size)
	status <- progress.EndTiming()
	return nil
}
