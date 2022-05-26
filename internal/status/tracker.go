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

// Tracker is used to track status when interacting with the target CAS.
// Note: This Tracker is a workaround. Implementation will be enhanced once
// oras-project/oras-go#150 is merged.
type Tracker struct {
	oras.Target
	out          io.Writer
	printLock    sync.Mutex
	printAfter   bool
	printExisted bool
	prompt       string
	verbose      bool
	*ManifestConfigOption
}

// NewPushTracker returns a new status tracking object for push command.
func NewPushTracker(target oras.Target, verbose bool) *Tracker {
	return &Tracker{
		Target:       target,
		out:          os.Stdout,
		prompt:       "Uploading",
		verbose:      verbose,
		printAfter:   true,
		printExisted: true,
	}
}

// NewPullTracker returns a new status tracking object for pull command.
func NewPullTracker(target oras.Target, option *ManifestConfigOption) *Tracker {
	return &Tracker{
		Target:               target,
		out:                  os.Stdout,
		prompt:               "Downloaded",
		verbose:              false,
		printAfter:           true,
		printExisted:         false,
		ManifestConfigOption: option,
	}
}

// ManifestConfigOption contains options for rewriting config
type ManifestConfigOption struct {
	Name      string
	MediaType string
}

// Push pushes the content, matching the expected descriptor with status tracking.
// Current implementation is a workaround before oras-go v2 supports copy
// option, see https://github.com/oras-project/oras-go/issues/59.
func (t *Tracker) Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) error {
	option := t.ManifestConfigOption
	if option != nil && option.MediaType == expected.MediaType && option.Name != "" {
		if expected.Annotations == nil {
			expected.Annotations = make(map[string]string)
		}
		expected.Annotations[ocispec.AnnotationTitle] = option.Name
	}

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
func (t *Tracker) Exists(ctx context.Context, target ocispec.Descriptor) (bool, error) {
	existed, err := t.Target.Exists(ctx, target)
	if t.printExisted && err == nil && existed {
		t.printLock.Lock()
		defer t.printLock.Unlock()
		fmt.Fprintln(t.out, digestString(target), "already exists")
	}
	return existed, err
}

// digestString gets the digest string from the descriptor for displaying
func digestString(desc ocispec.Descriptor) (digestString string) {
	digestString = desc.Digest.String()
	if err := desc.Digest.Validate(); err == nil {
		if algo := desc.Digest.Algorithm(); algo == digest.SHA256 {
			digestString = desc.Digest.Encoded()[:12]
		}
	}
	return digestString
}
