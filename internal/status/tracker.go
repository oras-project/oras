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

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
)

// StatusTracker is used to track status when interacting with the target CAS.
type StatusTracker struct {
	oras.Target
	out          io.Writer
	printLock    sync.Mutex
	printAfter   bool
	printExisted bool
	prompt       string
	verbose      bool
}

// NewPushTracker returns a new status tracking object.
func NewPushTracker(target oras.Target, verbose bool) *StatusTracker {
	return &StatusTracker{
		Target:       target,
		out:          os.Stdout,
		prompt:       "Uploading",
		verbose:      verbose,
		printAfter:   true,
		printExisted: true,
	}
}

// Push pushes a descriptor with status tracking.
// Current implementation is a workaround before oras-go v2 supports copy
// option, see https://github.com/oras-project/oras-go/issues/59.
func (t *StatusTracker) Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) error {
	print := func() {
		name, ok := expected.Annotations[ocispec.AnnotationTitle]
		if !ok {
			if !t.verbose {
				return
			}
			name = expected.MediaType
		}

		digestString := expected.Digest.String()
		if err := expected.Digest.Validate(); err == nil {
			if algo := expected.Digest.Algorithm(); algo == digest.SHA256 {
				digestString = expected.Digest.Encoded()[:12]
			}
		}
		t.printLock.Lock()
		defer t.printLock.Unlock()
		fmt.Fprintln(t.out, t.prompt, digestString, name)
	}

	if t.printAfter {
		if err := t.Target.Push(ctx, expected, content); err != nil {
			return err
		}
		print()
		return nil
	}

	print()
	return t.Target.Push(ctx, expected, content)
}

// Exists check if a descriptor exists in the store with status tracking.
// Current implementation is a workaround before oras-go v2 supports copy
// option, see https://github.com/oras-project/oras-go/issues/59.
func (t *StatusTracker) Exists(ctx context.Context, target ocispec.Descriptor) (bool, error) {
	existed, err := t.Target.Exists(ctx, target)
	if t.printExisted && err == nil && existed {
		t.printLock.Lock()
		defer t.printLock.Unlock()
		fmt.Fprintln(t.out, target.Digest.Encoded()[:12]+": Blob already exists")
	}
	return existed, err
}
