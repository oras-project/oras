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
)

// PullTracker is used to track status when pulling from the target CAS.
// Note: This PullTracker is a workaround. Implementation will be enhanced once
// oras-project/oras-go#150 is merged.
type PullTracker struct {
	oras.Target
	*ManifestConfigOption
	out       io.Writer
	printLock sync.Mutex
	prompt    string
	verbose   bool
	cache     oras.Target
}

// NewPullTracker returns a new status tracking object for pull command.
func NewPullTracker(target oras.Target, option *ManifestConfigOption, cache oras.Target) *PullTracker {
	return &PullTracker{
		Target:               target,
		out:                  os.Stdout,
		prompt:               "Downloaded",
		verbose:              false,
		ManifestConfigOption: option,
		cache:                cache,
	}
}

// ManifestConfigOption contains options for manifest config.
type ManifestConfigOption struct {
	Name      string
	MediaType string
}

// Push pushes the content, matching the expected descriptor with status
// tracking.
// Current implementation is a workaround before oras-go v2 supports copy
// option, see https://github.com/oras-project/oras-go/issues/59.
func (t *PullTracker) Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) error {
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

	src := t.Target
	if t.cache != nil {
		existed, err := t.cache.Exists(ctx, expected)
		if err != nil {
			return err
		}
		if !existed {
			if err := t.cache.Push(ctx, expected, content); err != nil {
				return err
			}
		}
		src = t.cache
	}
	if err := src.Push(ctx, expected, content); err != nil {
		return err
	}
	print()
	return nil
}
