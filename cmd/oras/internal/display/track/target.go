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
	"oras.land/oras/cmd/oras/internal/display/progress"
)

// Trackable can be tracked and supprots explicit prompting and stoping.
type Trackable interface {
	Prompt(desc ocispec.Descriptor, prompt string) error
	Close() error
}

// Target is a wrapper for oras.Target with tracked pushing.
type Target interface {
	oras.GraphTarget
	Trackable
}

type target struct {
	oras.Target
	manager      progress.Manager
	actionPrompt string
	donePrompt   string
}

func NewTarget(t oras.Target, actionPrompt, donePrompt string, tty *os.File) (Target, error) {
	manager, err := progress.NewManager(tty)
	if err != nil {
		return nil, err
	}

	return &target{
		Target:       t,
		manager:      manager,
		actionPrompt: actionPrompt,
		donePrompt:   donePrompt,
	}, nil
}

func (t *target) Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) error {
	r, err := managedReader(content, expected, t.manager, t.actionPrompt, t.donePrompt)
	if err != nil {
		return err
	}
	defer r.Stop()
	r.Start()
	if err := t.Target.Push(ctx, expected, r); err != nil {
		return err
	}

	r.status <- progress.EndTiming()
	r.status <- progress.NewStatus(t.donePrompt, expected, uint64(expected.Size))
	return nil
}

func (t *target) PushReference(ctx context.Context, expected ocispec.Descriptor, content io.Reader, reference string) error {
	r, err := managedReader(content, expected, t.manager, t.actionPrompt, t.donePrompt)
	if err != nil {
		return err
	}
	defer r.Stop()
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

	r.status <- progress.EndTiming()
	r.status <- progress.NewStatus(t.donePrompt, expected, uint64(expected.Size))
	return nil
}

func (t *target) Predecessors(ctx context.Context, node ocispec.Descriptor) ([]ocispec.Descriptor, error) {
	if p, ok := t.Target.(content.PredecessorFinder); ok {
		return p.Predecessors(ctx, node)
	}
	return nil, fmt.Errorf("target %v does not support Predecessors", reflect.TypeOf(t.Target))
}

func (t *target) Close() error {
	if err := t.manager.Close(); err != nil {
		return err
	}
	return nil
}

func (t *target) Prompt(desc ocispec.Descriptor, prompt string) error {
	status, err := t.manager.Add()
	if err != nil {
		return err
	}
	defer close(status)
	status <- progress.NewStatus(prompt, desc, uint64(desc.Size))
	status <- progress.EndTiming()
	return nil
}
