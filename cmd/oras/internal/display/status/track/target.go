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
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry"
	sprogress "oras.land/oras/cmd/oras/internal/display/status/progress"
	"oras.land/oras/internal/progress"
)

// GraphTarget is a tracked oras.GraphTarget.
type GraphTarget interface {
	oras.GraphTarget
	io.Closer
	Report(desc ocispec.Descriptor, state progress.State) error
}

type graphTarget struct {
	oras.GraphTarget
	manager progress.Manager
}

type referenceGraphTarget struct {
	*graphTarget
}

// NewTarget creates a new tracked Target.
func NewTarget(t oras.GraphTarget, prompts map[progress.State]string, tty *os.File) (GraphTarget, error) {
	manager, err := sprogress.NewManager(tty, prompts)
	if err != nil {
		return nil, err
	}
	gt := &graphTarget{
		GraphTarget: t,
		manager:     manager,
	}

	if _, ok := t.(registry.ReferencePusher); ok {
		return &referenceGraphTarget{
			graphTarget: gt,
		}, nil
	}
	return gt, nil
}

// Mount mounts a blob from a specified repository. This method is invoked only
// by the `*remote.Repository` target.
func (t *graphTarget) Mount(ctx context.Context, desc ocispec.Descriptor, fromRepo string, getContent func() (io.ReadCloser, error)) error {
	mounter := t.GraphTarget.(registry.Mounter)
	return mounter.Mount(ctx, desc, fromRepo, getContent)
}

// Push pushes the content to the base oras.GraphTarget with tracking.
func (t *graphTarget) Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) error {
	r, err := newReader(content, expected, t.manager)
	if err != nil {
		return err
	}
	defer r.StopTracker()
	if err := progress.Start(r.Tracker()); err != nil {
		return err
	}
	if err := t.GraphTarget.Push(ctx, expected, r); err != nil {
		if errors.Is(err, errdef.ErrAlreadyExists) {
			// allowed error types in oras-go oci and memory store
			if err := progress.Done(r.Tracker()); err != nil {
				return err
			}
		}
		return err
	}
	return progress.Done(r.Tracker())
}

// PushReference pushes the content to the base oras.GraphTarget with tracking.
func (rgt *referenceGraphTarget) PushReference(ctx context.Context, expected ocispec.Descriptor, content io.Reader, reference string) error {
	r, err := newReader(content, expected, rgt.manager)
	if err != nil {
		return err
	}
	defer r.StopTracker()
	if err := progress.Start(r.Tracker()); err != nil {
		return err
	}
	err = rgt.GraphTarget.(registry.ReferencePusher).PushReference(ctx, expected, r, reference)
	if err != nil {
		return err
	}
	return progress.Done(r.Tracker())
}

// Close closes the tracking manager.
func (t *graphTarget) Close() error {
	return t.manager.Close()
}

// Report prompts the user with the provided state and descriptor.
func (t *graphTarget) Report(desc ocispec.Descriptor, state progress.State) error {
	tracker, err := t.manager.Track(desc)
	if err != nil {
		return err
	}
	if err = tracker.Update(progress.Status{
		State:  state,
		Offset: desc.Size,
	}); err != nil {
		return err
	}
	return tracker.Close()
}
