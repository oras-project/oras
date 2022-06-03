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
	"os"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Struct writes status output.
type Status struct {
	lock    sync.Mutex
	verbose bool
	out     io.Writer
}

// NewStatus returns a new status struct.
func NewStatus(verbose bool) *Status {
	return &Status{
		verbose: verbose,
		out:     os.Stdout,
	}
}

// BeforeNodeCopied outputs status before a node got copied.
func (w *Status) BeforeNodeCopied(ctx context.Context, desc ocispec.Descriptor) error {
	name, ok := desc.Annotations[ocispec.AnnotationTitle]
	if !ok {
		if !w.verbose {
			return nil
		}
		name = desc.MediaType
	}
	return w.print("Uploading", ToShort(desc), name)
}

// OnCopySkipped outputs status when a node copy is skipped.
func (w *Status) OnCopySkipped(ctx context.Context, desc ocispec.Descriptor) error {
	return w.print("Existed ", ToShort(desc), desc.Annotations[ocispec.AnnotationTitle])
}

// print outputs status with locking.
func (w *Status) print(a ...any) error {
	w.lock.Lock()
	defer w.lock.Unlock()
	_, err := fmt.Fprintln(w.out, a...)
	return err
}
