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

package display

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras/cmd/oras/internal/display/track"
	"oras.land/oras/cmd/oras/internal/metadata"
	"oras.land/oras/cmd/oras/internal/option"
)

// PackHandler handles the pack output.
type PackHandler struct {
	template           string
	needTextOutput     bool
	verbose            bool
	fetcher            content.Fetcher
	tty                *os.File
	trackedGraphTarget track.GraphTarget
	committed          *sync.Map
	promptSkipped      string
	promptUploaded     string
	promptExists       string
	promptUploading    string
}

// NewPackHandler creates a new PackHandler.
func NewPackHandler(template string, tty *os.File, fetcher content.Fetcher, dst any, verbose bool) *PackHandler {
	ph := &PackHandler{
		template:        template,
		needTextOutput:  NeedTextOutput(template, tty),
		verbose:         verbose,
		fetcher:         fetcher,
		tty:             tty,
		committed:       &sync.Map{},
		promptSkipped:   "Skipped  ",
		promptUploaded:  "Uploaded ",
		promptExists:    "Exists   ",
		promptUploading: "Uploading",
	}
	if tracked, ok := dst.(track.GraphTarget); ok {
		ph.trackedGraphTarget = tracked
	}
	return ph
}

// OnCopySkipped provides display handler for skipping copying a blob/manifest.
func (ph *PackHandler) OnCopySkipped(ctx context.Context, desc ocispec.Descriptor) error {
	ph.committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
	if ph.trackedGraphTarget != nil {
		// TTY
		return ph.trackedGraphTarget.Prompt(desc, ph.promptExists)
	} else if ph.needTextOutput {
		return PrintStatus(desc, ph.promptExists, ph.verbose)
	}
	return nil
}

// PreCopy provides display handler before copying a blob/manifest.
func (ph *PackHandler) PreCopy(ctx context.Context, desc ocispec.Descriptor) error {
	if ph.trackedGraphTarget != nil {
		// TTY
		return nil
	} else if ph.needTextOutput {
		return PrintStatus(desc, ph.promptUploading, ph.verbose)
	}
	return nil
}

// PostCopy provides display handler after copying a blob/manifest.
func (ph *PackHandler) PostCopy(ctx context.Context, desc ocispec.Descriptor) error {
	ph.committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
	if ph.trackedGraphTarget != nil {
		// TTY
		return PrintSuccessorStatus(ctx, desc, ph.fetcher, ph.committed, func(d ocispec.Descriptor) error {
			return ph.trackedGraphTarget.Prompt(d, ph.promptSkipped)
		})
	} else if ph.needTextOutput {
		if err := PrintSuccessorStatus(ctx, desc, ph.fetcher, ph.committed, StatusPrinter(ph.promptSkipped, ph.verbose)); err != nil {
			return err
		}
		return PrintStatus(desc, ph.promptUploaded, ph.verbose)
	}
	return nil
}

// PostManifest provides display handler after pushing.
func (ph *PackHandler) PostPush(root ocispec.Descriptor, opts *option.Target, w io.Writer) error {
	if ph.needTextOutput {
		return Print("Pushed", opts.AnnotatedReference())
	}
	return option.WriteMetadata(ph.template, w, metadata.NewPush(root, opts.Path))
}

// PostAttach provides display handler after attaching.
func (ph *PackHandler) PostAttach(root, subject ocispec.Descriptor, opts *option.Target, w io.Writer) error {
	if ph.needTextOutput {
		digest := subject.Digest.String()
		if !strings.HasSuffix(opts.RawReference, digest) {
			opts.RawReference = fmt.Sprintf("%s@%s", opts.Path, subject.Digest)
		}
		Print("Attached to", opts.AnnotatedReference())
		Print("Digest:", root.Digest)
	}
	return option.WriteMetadata(ph.template, w, metadata.NewPush(root, opts.Path))
}

// Taggable returns a taggable with status printing.
func (ph *PackHandler) Taggable(t oras.Target) oras.Target {
	if ph.trackedGraphTarget != nil {
		// TTY
		t = ph.trackedGraphTarget.Inner()
	}
	if !ph.needTextOutput {
		return t
	}
	return NewTagStatusPrinter(t)
}
