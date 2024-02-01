package status

import (
	"context"
	"os"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras/cmd/oras/internal/display/status/track"
)

type TTYPushHandler struct {
	tty     *os.File
	tracked track.GraphTarget
}

func NewTTYPushHandler(tty *os.File) PushHandler {
	return &TTYPushHandler{
		tty: tty,
	}
}

func (ph *TTYPushHandler) OnFileLoading(name string) error {
	return nil
}

func (ph *TTYPushHandler) OnEmptyArtifact() error {
	return nil
}

func (ph *TTYPushHandler) TrackTarget(gt oras.GraphTarget) (oras.GraphTarget, error) {
	const (
		promptUploaded  = "Uploaded "
		promptUploading = "Uploading"
	)
	tracked, err := track.NewTarget(gt, promptUploading, promptUploaded, ph.tty)
	if err != nil {
		return nil, err
	}
	ph.tracked = tracked
	return tracked, nil
}

func (ph *TTYPushHandler) UpdateCopyOptions(opts *oras.CopyGraphOptions, fetcher content.Fetcher) {
	const (
		promptSkipped = "Skipped  "
		promptExists  = "Exists   "
	)
	committed := &sync.Map{}
	opts.OnCopySkipped = func(ctx context.Context, desc ocispec.Descriptor) error {
		committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
		return ph.tracked.Prompt(desc, promptExists)
	}
	opts.PostCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		committed.Store(desc.Digest.String(), desc.Annotations[ocispec.AnnotationTitle])
		return PrintSuccessorStatus(ctx, desc, fetcher, committed, func(d ocispec.Descriptor) error {
			return ph.tracked.Prompt(d, promptSkipped)
		})
	}
}

func NewTTYAttachHandler(tty *os.File) AttachHandler {
	return NewTTYPushHandler(tty)
}
