package main

import (
	"context"
	"fmt"
	"io"
	"sync"

	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
)

type statusTracker struct {
	oras.Target
	out     io.Writer
	lock    sync.Mutex
	prompt  string
	verbose bool
}

func (t *statusTracker) Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) error {
	if err := t.Target.Push(ctx, expected, content); err != nil {
		return err
	}

	name, ok := expected.Annotations[ocispec.AnnotationTitle]
	if !ok {
		if !t.verbose {
			return nil
		}
		name = expected.MediaType
	}

	digestString := expected.Digest.String()
	if err := expected.Digest.Validate(); err == nil {
		if algo := expected.Digest.Algorithm(); algo == digest.SHA256 {
			digestString = expected.Digest.Encoded()[:12]
		}
	}
	t.lock.Lock()
	defer t.lock.Unlock()
	fmt.Fprintln(t.out, t.prompt, digestString, name)

	return nil
}
