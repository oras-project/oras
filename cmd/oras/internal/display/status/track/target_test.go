//go:build freebsd || linux || netbsd || openbsd || solaris

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
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras/internal/testutils"
)

type testReferenceGraphTarget struct {
	oras.GraphTarget
}

func (t *testReferenceGraphTarget) PushReference(ctx context.Context, expected ocispec.Descriptor, content io.Reader, reference string) error {
	err := t.GraphTarget.Push(ctx, expected, content)
	if err != nil {
		return err
	}
	return t.GraphTarget.Tag(ctx, expected, reference)
}

func Test_referenceGraphTarget_PushReference(t *testing.T) {
	// prepare
	pty, device, err := testutils.NewPty()
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()
	src := memory.New()
	content := []byte("test")
	r := bytes.NewReader(content)
	desc := ocispec.Descriptor{
		MediaType: "application/octet-stream",
		Digest:    digest.FromBytes(content),
		Size:      int64(len(content)),
	}
	// test
	tag := "tagged"
	actionPrompt := "action"
	donePrompt := "done"
	target, err := NewTarget(&testReferenceGraphTarget{src}, actionPrompt, donePrompt, device)
	if err != nil {
		t.Fatal(err)
	}
	if rgt, ok := target.(*referenceGraphTarget); ok {
		if err := rgt.PushReference(context.Background(), desc, r, tag); err != nil {
			t.Fatal(err)
		}
		if err := rgt.manager.Close(); err != nil {
			t.Fatal(err)
		}
	} else {
		t.Fatal("not testing based on a referenceGraphTarget")
	}
	// validate
	if err = testutils.MatchPty(pty, device, donePrompt, desc.MediaType, "100.00%", desc.Digest.String()); err != nil {
		t.Fatal(err)
	}
}

func Test_referenceGraphTarget_Mount(t *testing.T) {
	target := graphTarget{GraphTarget: &remote.Repository{}}
	_ = target.Mount(context.Background(), ocispec.Descriptor{}, "", nil)
}

func Test_graphTarget_Push_alreadyExists(t *testing.T) {
	// prepare
	pty, device, err := testutils.NewPty()
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()
	src := memory.New()
	content := []byte("test")
	r := bytes.NewReader(content)
	desc := ocispec.Descriptor{
		MediaType: "application/octet-stream",
		Digest:    digest.FromBytes(content),
		Size:      int64(len(content)),
	}
	if err := src.Push(context.Background(), desc, r); err != nil {
		t.Fatal("Failed to prepare test environment:", err)
	}
	// test
	actionPrompt := "action"
	donePrompt := "done"
	target, err := NewTarget(src, actionPrompt, donePrompt, device)
	if err != nil {
		t.Fatal(err)
	}
	if gt, ok := target.(*graphTarget); ok {
		if err := gt.Push(context.Background(), desc, r); !errors.Is(err, errdef.ErrAlreadyExists) {
			t.Fatal("Expected ErrAlreadyExists, got:", err)
		}
		if err := gt.manager.Close(); err != nil {
			t.Fatal(err)
		}
	} else {
		t.Fatal("not testing based on a referenceGraphTarget")
	}
	// validate
	if err = testutils.MatchPty(pty, device, donePrompt, desc.MediaType, "100.00%", desc.Digest.String()); err != nil {
		t.Fatal(err)
	}
}
