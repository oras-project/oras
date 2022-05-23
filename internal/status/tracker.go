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

type StatusTracker struct {
	oras.Target
	out          io.Writer
	printLock    sync.Mutex
	printAfter   bool
	printExisted bool
	prompt       string
	verbose      bool
}

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

func (t *StatusTracker) Exists(ctx context.Context, target ocispec.Descriptor) (bool, error) {
	existed, err := t.Target.Exists(ctx, target)
	if t.printExisted && err == nil && existed {
		t.printLock.Lock()
		defer t.printLock.Unlock()
		fmt.Fprintln(t.out, target.Digest.Encoded()[:12]+": Blob already exists")
	}
	return existed, err
}
