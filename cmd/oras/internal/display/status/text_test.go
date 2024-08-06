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
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras/cmd/oras/internal/output"
	"oras.land/oras/internal/testutils"
)

var (
	ctx          context.Context
	builder      *strings.Builder
	printer      *output.Printer
	bogus        ocispec.Descriptor
	memStore     *memory.Store
	layerDesc    ocispec.Descriptor
	manifestDesc ocispec.Descriptor
	wantedError  = fmt.Errorf("wanted error")
)

func TestMain(m *testing.M) {
	// memory store for testing
	memStore = memory.New()
	layerContent := []byte("test")
	r := bytes.NewReader(layerContent)
	layerDesc = ocispec.Descriptor{
		MediaType: "application/octet-stream",
		Digest:    digest.FromBytes(layerContent),
		Size:      int64(len(layerContent)),
	}
	if err := memStore.Push(context.Background(), layerDesc, r); err != nil {
		fmt.Println("Setup failed:", err)
		os.Exit(1)
	}
	if err := memStore.Tag(context.Background(), layerDesc, layerDesc.Digest.String()); err != nil {
		fmt.Println("Setup failed:", err)
		os.Exit(1)
	}

	layer1Desc := layerDesc
	layer1Desc.Annotations = map[string]string{ocispec.AnnotationTitle: "layer1"}
	layer2Desc := layerDesc
	layer2Desc.Annotations = map[string]string{ocispec.AnnotationTitle: "layer2"}
	manifest := ocispec.Manifest{
		MediaType: ocispec.MediaTypeImageManifest,
		Layers:    []ocispec.Descriptor{layer1Desc, layer2Desc},
		Config:    layerDesc,
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
	if err := memStore.Tag(context.Background(), layerDesc, layerDesc.Digest.String()); err != nil {
		fmt.Println("Setup failed:", err)
		os.Exit(1)
	}

	ctx = context.Background()
	builder = &strings.Builder{}
	printer = output.NewPrinter(builder, os.Stderr, false)
	bogus = ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest}
	m.Run()
}

func TestTextCopyHandler_OnMounted(t *testing.T) {
	fetcher := testutils.NewMockFetcher(t)
	ch := NewTextCopyHandler(printer, fetcher.Fetcher)
	if ch.OnMounted(ctx, fetcher.OciImage) != nil {
		t.Error("OnMounted() should not return an error")
	}

}

func TestTextCopyHandler_OnCopySkipped(t *testing.T) {
	fetcher := testutils.NewMockFetcher(t)
	ch := NewTextCopyHandler(printer, fetcher.Fetcher)
	if ch.OnCopySkipped(ctx, fetcher.OciImage) != nil {
		t.Error("OnCopySkipped() should not return an error")
	}
}

func TestTextCopyHandler_PostCopy(t *testing.T) {
	fetcher := testutils.NewMockFetcher(t)
	ch := NewTextCopyHandler(printer, fetcher.Fetcher)
	if ch.PostCopy(ctx, fetcher.OciImage) != nil {
		t.Error("PostCopy() should not return an error")
	}
	if ch.PostCopy(ctx, bogus) == nil {
		t.Error("PostCopy() should return an error")
	}
}

func TestTextCopyHandler_PreCopy(t *testing.T) {
	fetcher := testutils.NewMockFetcher(t)
	ch := NewTextCopyHandler(printer, fetcher.Fetcher)
	if ch.PreCopy(ctx, fetcher.OciImage) != nil {
		t.Error("PreCopy() should not return an error")
	}
}
