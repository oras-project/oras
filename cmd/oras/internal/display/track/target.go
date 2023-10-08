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
	"fmt"
	"io"
	"os"
	"reflect"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/display/progress"
)

// Trackable can be tracked and supprots explicit prompting and stoping.
type Target struct {
	oras.Target
	manager      progress.Manager
	actionPrompt string
	donePrompt   string
}

func NewTarget(t oras.Target, actionPrompt, donePrompt string, tty *os.File) (*Target, error) {
	manager, err := progress.NewManager(tty)
	if err != nil {
		return nil, err
	}

	return &Target{
		Target:       t,
		manager:      manager,
		actionPrompt: actionPrompt,
		donePrompt:   donePrompt,
	}, nil
}

func (t *Target) Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) error {
	r, err := managedReader(content, expected, t.manager, t.actionPrompt, t.donePrompt)
	if err != nil {
		return err
	}
	defer r.Close()
	r.Start()
	if err := t.Target.Push(ctx, expected, r); err != nil {
		return err
	}
	r.Done()
	return nil
}

func (t *Target) PushReference(ctx context.Context, expected ocispec.Descriptor, content io.Reader, reference string) error {
	r, err := managedReader(content, expected, t.manager, t.actionPrompt, t.donePrompt)
	if err != nil {
		return err
	}
	defer r.Close()
	r.Start()
	if rp, ok := t.Target.(registry.ReferencePusher); ok {
		err = rp.PushReference(ctx, expected, r, reference)
	} else {
		if err := t.Target.Push(ctx, expected, r); err != nil {
			return err
		}
		err = t.Target.Tag(ctx, expected, reference)
	}
	if err != nil {
		return err
	}
	r.Done()
	return nil
}

func (t *Target) Predecessors(ctx context.Context, node ocispec.Descriptor) ([]ocispec.Descriptor, error) {
	if p, ok := t.Target.(content.PredecessorFinder); ok {
		return p.Predecessors(ctx, node)
	}
	return nil, fmt.Errorf("Target %v does not support Predecessors", reflect.TypeOf(t.Target))
}

// Close closes the Target to stop tracking.
func (t *Target) Close() error {
	return t.manager.Close()
}

// Prompt prompts the user with the provided prompt and descriptor.
// If Target is not set, only prints status.
func (t *Target) Prompt(desc ocispec.Descriptor, prompt string, verbose bool) error {
	if t == nil {
		display.PrintStatus(desc, prompt, verbose)
		return nil
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
