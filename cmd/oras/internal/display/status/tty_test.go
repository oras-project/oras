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
	"fmt"
	"os"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras/cmd/oras/internal/display/status/console/testutils"
	"oras.land/oras/cmd/oras/internal/display/status/track"
)

var (
	memStore        *memory.Store
	memDesc         ocispec.Descriptor
	manifestConent  = []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","artifactType":"application/vnd.unknown.artifact.v1","config":{"mediaType":"application/vnd.oci.empty.v1+json","digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2,"data":"e30="},"layers":[{"mediaType":"application/vnd.oci.empty.v1+json","digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2,"data":"e30="}]}`)
	manifestDigest  = "sha256:1bb053792feb8d8d590001c212f2defad9277e091d2aa868cde2879ff41abb1b"
	configContent   = []byte("{}")
	configDigest    = "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a"
	configMediaType = "application/vnd.oci.empty.v1+json"
	host            string
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
	_, err = ph.TrackTarget(store)
	if err != nil {
		t.Error("TrackTarget() should not return an error")
	}
	if ttyPushHandler, ok := ph.(*TTYPushHandler); !ok {
		t.Errorf("TrackTarget() should return a *TTYPushHandler, got %T", ttyPushHandler)
	} else if ttyPushHandler.tracked.Inner() != store {
		t.Errorf("TrackTarget() tracks unexpected tracked target: %T", ttyPushHandler.tracked)
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
	gt, err := ph.TrackTarget(memory.New())
	if err != nil {
		t.Errorf("TrackTarget() should not return an error: %v", err)
	}
	// test copy
	opts := oras.CopyGraphOptions{}
	ph.UpdateCopyOptions(&opts, memStore)
	if err := oras.CopyGraph(context.Background(), memStore, gt, memDesc, opts); err != nil {
		t.Errorf("CopyGraph() should not return an error: %v", err)
	}
	if tracked, ok := gt.(track.GraphTarget); !ok {
		t.Errorf("TrackTarget() should return a *track.GraphTarget, got %T", tracked)
	} else {
		tracked.Close()
	}
	// validate
	if err = testutils.MatchPty(pty, slave, "Uploaded", memDesc.MediaType, "100.00%", memDesc.Digest.String()); err != nil {
		t.Fatal(err)
	}
}
