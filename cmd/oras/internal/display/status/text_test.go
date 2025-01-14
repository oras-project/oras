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
	"sync"
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
	printer = output.NewPrinter(builder, os.Stderr)
	bogus = ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest}
	os.Exit(m.Run())
}

func validatePrinted(t *testing.T, expected string) {
	actual := strings.TrimSpace(builder.String())
	if expected != actual {
		t.Error("Output does not match expected <" + expected + "> actual <" + actual + ">")
	}
}

func TestTextCopyHandler_OnMounted(t *testing.T) {
	builder.Reset()
	ch := NewTextCopyHandler(printer, mockFetcher.Fetcher)
	if ch.OnMounted(ctx, mockFetcher.OciImage) != nil {
		t.Error("OnMounted() should not return an error")
	}
	validatePrinted(t, "Mounted 0b442c23c1dd oci-image")
}

func TestTextCopyHandler_OnCopySkipped(t *testing.T) {
	builder.Reset()
	ch := NewTextCopyHandler(printer, mockFetcher.Fetcher)
	if ch.OnCopySkipped(ctx, mockFetcher.OciImage) != nil {
		t.Error("OnCopySkipped() should not return an error")
	}
	validatePrinted(t, "Exists  0b442c23c1dd oci-image")
}

func TestTextCopyHandler_PostCopy_titled(t *testing.T) {
	builder.Reset()
	ch := NewTextCopyHandler(printer, mockFetcher.Fetcher)
	if ch.PostCopy(ctx, mockFetcher.OciImage) != nil {
		t.Error("PostCopy() should not return an error")
	}
	if ch.PostCopy(ctx, bogus) == nil {
		t.Error("PostCopy() should return an error")
	}
	validatePrinted(t, "Copied  0b442c23c1dd oci-image")
}

func TestTextCopyHandler_PostCopy_skipped(t *testing.T) {
	builder.Reset()
	ch := &TextCopyHandler{
		printer:   printer,
		fetcher:   mockFetcher.Fetcher,
		committed: &sync.Map{},
	}
	ch.committed.Store(mockFetcher.ImageLayer.Digest.String(), mockFetcher.ImageLayer.Annotations[ocispec.AnnotationTitle]+"bogus")
	if err := ch.PostCopy(ctx, mockFetcher.OciImage); err != nil {
		t.Error("PostCopy() returns unexpected err:", err)
	}
	validatePrinted(t, "Skipped f6b87e8e0fe1 layer\nCopied  0b442c23c1dd oci-image")
}

func TestTextCopyHandler_PreCopy(t *testing.T) {
	builder.Reset()
	ch := NewTextCopyHandler(printer, mockFetcher.Fetcher)
	if ch.PreCopy(ctx, mockFetcher.OciImage) != nil {
		t.Error("PreCopy() should not return an error")
	}
	validatePrinted(t, "Copying 0b442c23c1dd oci-image")
}

func TestTextPullHandler_OnNodeDownloaded(t *testing.T) {
	builder.Reset()
	ph := NewTextPullHandler(printer)
	if ph.OnNodeDownloaded(mockFetcher.OciImage) != nil {
		t.Error("OnNodeDownloaded() should not return an error")
	}
	validatePrinted(t, "Downloaded  0b442c23c1dd oci-image")
}

func TestTextPullHandler_OnNodeDownloading(t *testing.T) {
	builder.Reset()
	ph := NewTextPullHandler(printer)
	if ph.OnNodeDownloading(mockFetcher.OciImage) != nil {
		t.Error("OnNodeDownloading() should not return an error")
	}
	validatePrinted(t, "Downloading 0b442c23c1dd oci-image")
}

func TestTextPullHandler_OnNodeProcessing(t *testing.T) {
	builder.Reset()
	ph := NewTextPullHandler(printer)
	if ph.OnNodeProcessing(mockFetcher.OciImage) != nil {
		t.Error("OnNodeProcessing() should not return an error")
	}
	validatePrinted(t, "Processing  0b442c23c1dd oci-image")
}

func TestTextPullHandler_OnNodeRestored(t *testing.T) {
	builder.Reset()
	ph := NewTextPullHandler(printer)
	if ph.OnNodeRestored(mockFetcher.OciImage) != nil {
		t.Error("OnNodeRestored() should not return an error")
	}
	validatePrinted(t, "Restored    0b442c23c1dd oci-image")
}

func TestTextPullHandler_OnNodeSkipped(t *testing.T) {
	builder.Reset()
	ph := NewTextPullHandler(printer)
	if ph.OnNodeSkipped(mockFetcher.OciImage) != nil {
		t.Error("OnNodeSkipped() should not return an error")
	}
	validatePrinted(t, "Skipped     0b442c23c1dd oci-image")
}

func TestTextPushHandler_OnCopySkipped(t *testing.T) {
	builder.Reset()
	ph := NewTextPushHandler(printer, mockFetcher.Fetcher)
	if ph.OnCopySkipped(ctx, mockFetcher.OciImage) != nil {
		t.Error("OnCopySkipped() should not return an error")
	}
	validatePrinted(t, "Exists    0b442c23c1dd oci-image")
}

func TestTextPushHandler_OnEmptyArtifact(t *testing.T) {
	builder.Reset()
	ph := NewTextPushHandler(printer, mockFetcher.Fetcher)
	if ph.OnEmptyArtifact() != nil {
		t.Error("OnEmptyArtifact() should not return an error")
	}
	validatePrinted(t, "Uploading empty artifact")
}

func TestTextPushHandler_OnFileLoading(t *testing.T) {
	builder.Reset()
	ph := NewTextPushHandler(printer, mockFetcher.Fetcher)
	if ph.OnFileLoading("name") != nil {
		t.Error("OnFileLoading() should not return an error")
	}
	validatePrinted(t, "")
}

func TestTextPushHandler_PostCopy(t *testing.T) {
	builder.Reset()
	ph := NewTextPushHandler(printer, mockFetcher.Fetcher)
	if ph.PostCopy(ctx, mockFetcher.OciImage) != nil {
		t.Error("PostCopy() should not return an error")
	}
	validatePrinted(t, "Uploaded  0b442c23c1dd oci-image")
}

func TestTextPushHandler_PreCopy(t *testing.T) {
	builder.Reset()
	ph := NewTextPushHandler(printer, mockFetcher.Fetcher)
	if ph.PreCopy(ctx, mockFetcher.OciImage) != nil {
		t.Error("PreCopy() should not return an error")
	}
	validatePrinted(t, "Uploading 0b442c23c1dd oci-image")
}

func TestTextManifestPushHandler_OnPushSkipped(t *testing.T) {
	mph := NewTextManifestPushHandler(printer, ocispec.Descriptor{})
	if mph.OnManifestPushSkipped() != nil {
		t.Error("OnManifestExists() should not return an error")
	}
}

func TestTextManifestIndexUpdateHandler_OnManifestAdded(t *testing.T) {
	tests := []struct {
		name    string
		printer *output.Printer
		ref     string
		desc    ocispec.Descriptor
		wantErr bool
	}{
		{
			name:    "ref is a digest",
			printer: output.NewPrinter(os.Stdout, os.Stderr),
			ref:     "sha256:fd6ed2f36b5465244d5dc86cb4e7df0ab8a9d24adc57825099f522fe009a22bb",
			desc:    ocispec.Descriptor{MediaType: "test", Digest: "sha256:fd6ed2f36b5465244d5dc86cb4e7df0ab8a9d24adc57825099f522fe009a22bb", Size: 25},
			wantErr: false,
		},
		{
			name:    "ref is not a digest",
			printer: output.NewPrinter(os.Stdout, os.Stderr),
			ref:     "v1",
			desc:    ocispec.Descriptor{MediaType: "test", Digest: "sha256:fd6ed2f36b5465244d5dc86cb4e7df0ab8a9d24adc57825099f522fe009a22bb", Size: 25},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			miuh := &TextManifestIndexUpdateHandler{
				printer: tt.printer,
			}
			if err := miuh.OnManifestAdded(tt.ref, tt.desc); (err != nil) != tt.wantErr {
				t.Errorf("TextManifestIndexUpdateHandler.OnManifestAdded() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTextManifestIndexUpdateHandler_OnIndexMerged(t *testing.T) {
	tests := []struct {
		name    string
		printer *output.Printer
		ref     string
		desc    ocispec.Descriptor
		wantErr bool
	}{
		{
			name:    "ref is a digest",
			printer: output.NewPrinter(os.Stdout, os.Stderr),
			ref:     "sha256:fd6ed2f36b5465244d5dc86cb4e7df0ab8a9d24adc57825099f522fe009a22bb",
			desc:    ocispec.Descriptor{MediaType: "test", Digest: "sha256:fd6ed2f36b5465244d5dc86cb4e7df0ab8a9d24adc57825099f522fe009a22bb", Size: 25},
			wantErr: false,
		},
		{
			name:    "ref is not a digest",
			printer: output.NewPrinter(os.Stdout, os.Stderr),
			ref:     "v1",
			desc:    ocispec.Descriptor{MediaType: "test", Digest: "sha256:fd6ed2f36b5465244d5dc86cb4e7df0ab8a9d24adc57825099f522fe009a22bb", Size: 25},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			miuh := &TextManifestIndexUpdateHandler{
				printer: tt.printer,
			}
			if err := miuh.OnIndexMerged(tt.ref, tt.desc); (err != nil) != tt.wantErr {
				t.Errorf("TextManifestIndexUpdateHandler.OnIndexMerged() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
