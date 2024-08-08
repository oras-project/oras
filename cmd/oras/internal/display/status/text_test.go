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
	"os"
	"strings"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/output"
	"oras.land/oras/internal/testutils"
)

var (
	ctx         context.Context
	builder     *strings.Builder
	printer     *output.Printer
	bogus       ocispec.Descriptor
	mockFetcher testutils.MockFetcher
)

func TestMain(m *testing.M) {
	mockFetcher = testutils.NewMockFetcher()
	ctx = context.Background()
	builder = &strings.Builder{}
	printer = output.NewPrinter(builder, os.Stderr, false)
	bogus = ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest}
	m.Run()
}

func TestTextCopyHandler_OnMounted(t *testing.T) {
	defer builder.Reset()
	expected := "Mounted 6f42718876ce oci-image"
	ch := NewTextCopyHandler(printer, mockFetcher.Fetcher)
	if ch.OnMounted(ctx, mockFetcher.OciImage) != nil {
		t.Error("OnMounted() should not return an error")
	}
	actual := strings.TrimSpace(builder.String())
	if expected != actual {
		t.Error("Output does not match expected <" + expected + "> actual <" + actual + ">")
	}
}

func TestTextCopyHandler_OnCopySkipped(t *testing.T) {
	defer builder.Reset()
	expected := "Exists  6f42718876ce oci-image"
	ch := NewTextCopyHandler(printer, mockFetcher.Fetcher)
	if ch.OnCopySkipped(ctx, mockFetcher.OciImage) != nil {
		t.Error("OnCopySkipped() should not return an error")
	}
	actual := strings.TrimSpace(builder.String())
	if expected != actual {
		t.Error("Output does not match expected <" + expected + "> actual <" + actual + ">")
	}
}

func TestTextCopyHandler_PostCopy(t *testing.T) {
	defer builder.Reset()
	expected := "Copied  6f42718876ce oci-image"
	ch := NewTextCopyHandler(printer, mockFetcher.Fetcher)
	if ch.PostCopy(ctx, mockFetcher.OciImage) != nil {
		t.Error("PostCopy() should not return an error")
	}
	if ch.PostCopy(ctx, bogus) == nil {
		t.Error("PostCopy() should return an error")
	}
	actual := strings.TrimSpace(builder.String())
	if expected != actual {
		t.Error("Output does not match expected <" + expected + "> actual <" + actual + ">")
	}
}

func TestTextCopyHandler_PreCopy(t *testing.T) {
	defer builder.Reset()
	expected := "Copying 6f42718876ce oci-image"
	ch := NewTextCopyHandler(printer, mockFetcher.Fetcher)
	if ch.PreCopy(ctx, mockFetcher.OciImage) != nil {
		t.Error("PreCopy() should not return an error")
	}
	actual := strings.TrimSpace(builder.String())
	if expected != actual {
		t.Error("Output does not match expected <" + expected + "> actual <" + actual + ">")
	}
}

func TestTextPullHandler_OnNodeDownloaded(t *testing.T) {
	defer builder.Reset()
	expected := "Downloaded  6f42718876ce oci-image"
	ph := NewTextPullHandler(printer)
	if ph.OnNodeDownloaded(mockFetcher.OciImage) != nil {
		t.Error("OnNodeDownloaded() should not return an error")
	}
	actual := strings.TrimSpace(builder.String())
	if expected != actual {
		t.Error("Output does not match expected <" + expected + "> actual <" + actual + ">")
	}
}

func TestTextPullHandler_OnNodeDownloading(t *testing.T) {
	defer builder.Reset()
	expected := "Downloading 6f42718876ce oci-image"
	ph := NewTextPullHandler(printer)
	if ph.OnNodeDownloading(mockFetcher.OciImage) != nil {
		t.Error("OnNodeDownloading() should not return an error")
	}
	actual := strings.TrimSpace(builder.String())
	if expected != actual {
		t.Error("Output does not match expected <" + expected + "> actual <" + actual + ">")
	}
}

func TestTextPullHandler_OnNodeProcessing(t *testing.T) {
	defer builder.Reset()
	expected := "Processing  6f42718876ce oci-image"
	ph := NewTextPullHandler(printer)
	if ph.OnNodeProcessing(mockFetcher.OciImage) != nil {
		t.Error("OnNodeProcessing() should not return an error")
	}
	actual := strings.TrimSpace(builder.String())
	if expected != actual {
		t.Error("Output does not match expected <" + expected + "> actual <" + actual + ">")
	}
}

func TestTextPullHandler_OnNodeRestored(t *testing.T) {
	defer builder.Reset()
	expected := "Restored    6f42718876ce oci-image"
	ph := NewTextPullHandler(printer)
	if ph.OnNodeRestored(mockFetcher.OciImage) != nil {
		t.Error("OnNodeRestored() should not return an error")
	}
	actual := strings.TrimSpace(builder.String())
	if expected != actual {
		t.Error("Output does not match expected <" + expected + "> actual <" + actual + ">")
	}
}

func TestTextPullHandler_OnNodeSkipped(t *testing.T) {
	defer builder.Reset()
	expected := "Skipped     6f42718876ce oci-image"
	ph := NewTextPullHandler(printer)
	if ph.OnNodeSkipped(mockFetcher.OciImage) != nil {
		t.Error("OnNodeSkipped() should not return an error")
	}
	actual := strings.TrimSpace(builder.String())
	if expected != actual {
		t.Error("Output does not match expected <" + expected + "> actual <" + actual + ">")
	}
}

func TestTextPushHandler_OnCopySkipped(t *testing.T) {
	defer builder.Reset()
	expected := "Exists    6f42718876ce oci-image"
	ph := NewTextPushHandler(printer, mockFetcher.Fetcher)
	if ph.OnCopySkipped(ctx, mockFetcher.OciImage) != nil {
		t.Error("OnCopySkipped() should not return an error")
	}
	actual := strings.TrimSpace(builder.String())
	if expected != actual {
		t.Error("Output does not match expected <" + expected + "> actual <" + actual + ">")
	}
}

func TestTextPushHandler_OnEmptyArtifact(t *testing.T) {
	defer builder.Reset()
	expected := "Uploading empty artifact"
	ph := NewTextPushHandler(printer, mockFetcher.Fetcher)
	if ph.OnEmptyArtifact() != nil {
		t.Error("OnEmptyArtifact() should not return an error")
	}
	actual := strings.TrimSpace(builder.String())
	if expected != actual {
		t.Error("Output does not match expected <" + expected + "> actual <" + actual + ">")
	}
}

func TestTextPushHandler_OnFileLoading(t *testing.T) {
	defer builder.Reset()
	expected := ""
	ph := NewTextPushHandler(printer, mockFetcher.Fetcher)
	if ph.OnFileLoading("name") != nil {
		t.Error("OnFileLoading() should not return an error")
	}
	actual := strings.TrimSpace(builder.String())
	if expected != actual {
		t.Error("Output does not match expected <" + expected + "> actual <" + actual + ">")
	}
}

func TestTextPushHandler_PostCopy(t *testing.T) {
	defer builder.Reset()
	expected := "Uploaded  6f42718876ce oci-image"
	ph := NewTextPushHandler(printer, mockFetcher.Fetcher)
	if ph.PostCopy(ctx, mockFetcher.OciImage) != nil {
		t.Error("PostCopy() should not return an error")
	}
	actual := strings.TrimSpace(builder.String())
	if expected != actual {
		t.Error("Output does not match expected <" + expected + "> actual <" + actual + ">")
	}
}

func TestTextPushHandler_PreCopy(t *testing.T) {
	defer builder.Reset()
	expected := "Uploading 6f42718876ce oci-image"
	ph := NewTextPushHandler(printer, mockFetcher.Fetcher)
	if ph.PreCopy(ctx, mockFetcher.OciImage) != nil {
		t.Error("PreCopy() should not return an error")
	}
	actual := strings.TrimSpace(builder.String())
	if expected != actual {
		t.Error("Output does not match expected <" + expected + "> actual <" + actual + ">")
	}
}
