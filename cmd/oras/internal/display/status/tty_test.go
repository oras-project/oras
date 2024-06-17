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

package status

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras/cmd/oras/internal/display/status/console/testutils"
	"oras.land/oras/cmd/oras/internal/display/status/track"
)

var (
	memStore     *memory.Store
	memDesc      ocispec.Descriptor
	manifestDesc ocispec.Descriptor
)

func TestMain(m *testing.M) {
	// memory store for testing
	memStore = memory.New()
	content := []byte("test")
	r := bytes.NewReader(content)
	memDesc = ocispec.Descriptor{
		MediaType: "application/octet-stream",
		Digest:    digest.FromBytes(content),
		Size:      int64(len(content)),
	}
	if err := memStore.Push(context.Background(), memDesc, r); err != nil {
		fmt.Println("Setup failed:", err)
		os.Exit(1)
	}
	if err := memStore.Tag(context.Background(), memDesc, memDesc.Digest.String()); err != nil {
		fmt.Println("Setup failed:", err)
		os.Exit(1)
	}

	layer1Desc := memDesc
	layer1Desc.Annotations = map[string]string{ocispec.AnnotationTitle: "layer1"}
	layer2Desc := memDesc
	layer2Desc.Annotations = map[string]string{ocispec.AnnotationTitle: "layer2"}
	manifest := ocispec.Manifest{
		MediaType: ocispec.MediaTypeImageManifest,
		Layers:    []ocispec.Descriptor{layer1Desc, layer2Desc},
	}
	manifestContent, err := json.Marshal(&manifest)
	if err != nil {
		fmt.Println("Setup failed:", err)
		os.Exit(1)
	}
	manifestDesc = ocispec.Descriptor{
		MediaType: manifest.MediaType,
		Size:      int64(len(manifestContent)),
		Digest:    digest.FromBytes(manifestContent),
	}
	if err := memStore.Push(context.Background(), manifestDesc, strings.NewReader(string(manifestContent))); err != nil {
		fmt.Println("Setup failed:", err)
		os.Exit(1)
	}
	if err := memStore.Tag(context.Background(), memDesc, memDesc.Digest.String()); err != nil {
		fmt.Println("Setup failed:", err)
		os.Exit(1)
	}
	m.Run()
}

func TestTTYPushHandler_OnFileLoading(t *testing.T) {
	ph := NewTTYPushHandler(os.Stdout)
	if ph.OnFileLoading("test") != nil {
		t.Error("OnFileLoading() should not return an error")
	}
}

func TestTTYPushHandler_OnEmptyArtifact(t *testing.T) {
	ph := NewTTYAttachHandler(os.Stdout)
	if ph.OnEmptyArtifact() != nil {
		t.Error("OnEmptyArtifact() should not return an error")
	}
}

func TestTTYPushHandler_TrackTarget(t *testing.T) {
	// prepare pty
	_, slave, err := testutils.NewPty()
	if err != nil {
		t.Fatal(err)
	}
	defer slave.Close()
	ph := NewTTYPushHandler(slave)
	store := memory.New()
	// test
	_, fn, err := ph.TrackTarget(store)
	if err != nil {
		t.Error("TrackTarget() should not return an error")
	}
	defer func() {
		if err := fn(); err != nil {
			t.Fatal(err)
		}
	}()
	if ttyPushHandler, ok := ph.(*TTYPushHandler); !ok {
		t.Errorf("TrackTarget() should return a *TTYPushHandler, got %T", ttyPushHandler)
	}
}

func TestTTYPushHandler_TrackTarget_invalidTTY(t *testing.T) {
	ph := NewTTYPushHandler(os.Stdin)
	if _, _, err := ph.TrackTarget(nil); err == nil {
		t.Error("TrackTarget() should return an error for non-tty file")
	}
}

func TestTTYPushHandler_UpdateCopyOptions(t *testing.T) {
	// prepare pty
	pty, slave, err := testutils.NewPty()
	if err != nil {
		t.Fatal(err)
	}
	defer slave.Close()
	ph := NewTTYPushHandler(slave)
	gt, _, err := ph.TrackTarget(memory.New())
	if err != nil {
		t.Errorf("TrackTarget() should not return an error: %v", err)
	}
	// test
	opts := oras.CopyGraphOptions{}
	ph.UpdateCopyOptions(&opts, memStore)
	if err := oras.CopyGraph(context.Background(), memStore, gt, memDesc, opts); err != nil {
		t.Errorf("CopyGraph() should not return an error: %v", err)
	}
	if err := oras.CopyGraph(context.Background(), memStore, gt, memDesc, opts); err != nil {
		t.Errorf("CopyGraph() should not return an error: %v", err)
	}
	if tracked, ok := gt.(track.GraphTarget); !ok {
		t.Errorf("TrackTarget() should return a *track.GraphTarget, got %T", tracked)
	} else {
		tracked.Close()
	}
	// validate
	if err = testutils.MatchPty(pty, slave, "Exists", memDesc.MediaType, "100.00%", memDesc.Digest.String()); err != nil {
		t.Fatal(err)
	}
}

func Test_TTYPullHandler_TrackTarget(t *testing.T) {
	src := memory.New()
	t.Run("has TTY", func(t *testing.T) {
		_, device, err := testutils.NewPty()
		if err != nil {
			t.Fatal(err)
		}
		defer device.Close()
		ph := NewTTYPullHandler(device)
		got, fn, err := ph.TrackTarget(src)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := fn(); err != nil {
				t.Fatal(err)
			}
		}()
		if got == src {
			t.Fatal("GraphTarget not be modified on TTY")
		}
	})

	t.Run("invalid TTY", func(t *testing.T) {
		ph := NewTTYPullHandler(nil)

		if _, _, err := ph.TrackTarget(src); err == nil {
			t.Fatal("expected error for no tty but got nil")
		}
	})
}

func TestTTYPullHandler_OnNodeDownloading(t *testing.T) {
	ph := NewTTYPullHandler(nil)
	if err := ph.OnNodeDownloading(ocispec.Descriptor{}); err != nil {
		t.Error("OnNodeDownloading() should not return an error")
	}
}

func TestTTYPullHandler_OnNodeDownloaded(t *testing.T) {
	ph := NewTTYPullHandler(nil)
	if err := ph.OnNodeDownloaded(ocispec.Descriptor{}); err != nil {
		t.Error("OnNodeDownloaded() should not return an error")
	}
}

func TestTTYPullHandler_OnNodeProcessing(t *testing.T) {
	ph := NewTTYPullHandler(nil)
	if err := ph.OnNodeProcessing(ocispec.Descriptor{}); err != nil {
		t.Error("OnNodeProcessing() should not return an error")
	}
}
