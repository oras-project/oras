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

// PushTracker is used to track status when pushing into the target CAS.
// Note: This PushTracker is a workaround. Implementation will be enhanced once
// oras-project/oras-go#150 is merged.
type PushTracker struct {
	oras.Target
	out          io.Writer
	printLock    sync.Mutex
	printExisted bool
	prompt       string
	verbose      bool
}

// NewPushTracker returns a new status tracking object for push command.
func NewPushTracker(target oras.Target, verbose bool) *PushTracker {
	return &PushTracker{
		Target:       target,
		out:          os.Stdout,
		prompt:       "Uploading",
		verbose:      verbose,
		printExisted: true,
	}
}

// Push pushes the content, matching the expected descriptor with status
// tracking.
// Current implementation is a workaround before oras-go v2 supports copy
// option, see https://github.com/oras-project/oras-go/issues/59.
func (t *PushTracker) Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) error {
	print := func() {
		name, ok := expected.Annotations[ocispec.AnnotationTitle]
		if !ok {
			if !t.verbose {
				return
			}
			name = expected.MediaType
		}
		t.printLock.Lock()
		defer t.printLock.Unlock()
		fmt.Fprintln(t.out, t.prompt, digestString(expected), name)
	}

	if err := t.Target.Push(ctx, expected, content); err != nil {
		return err
	}
	print()
	return nil
}

// Exists check if a descriptor exists in the store with status tracking.
// Current implementation is a workaround before oras-go v2 supports copy
// option, see https://github.com/oras-project/oras-go/issues/59.
func (t *PushTracker) Exists(ctx context.Context, target ocispec.Descriptor) (bool, error) {
	existed, err := t.Target.Exists(ctx, target)

	if t.printExisted && err == nil && existed {
		t.printLock.Lock()
		defer t.printLock.Unlock()
		fmt.Fprintln(t.out, "Existed  ", digestString(target), target.Annotations[ocispec.AnnotationTitle])
	}
	return existed, err
}

// digestString gets the digest string from the descriptor for displaying.
func digestString(desc ocispec.Descriptor) (digestString string) {
	digestString = desc.Digest.String()
	if err := desc.Digest.Validate(); err == nil {
		if algo := desc.Digest.Algorithm(); algo == digest.SHA256 {
			digestString = desc.Digest.Encoded()[:12]
		}
	}
	return digestString
}
