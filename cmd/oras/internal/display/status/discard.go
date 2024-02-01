package status

import (
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
)

type DiscardHandler struct{}

func NewDiscardHandler() DiscardHandler {
	return DiscardHandler{}
}

func (DiscardHandler) OnFileLoading(name string) error {
	return nil
}

func (DiscardHandler) OnEmptyArtifact() error {
	return nil
}

func (DiscardHandler) TrackTarget(gt oras.GraphTarget) (oras.GraphTarget, error) {
	return gt, nil
}

func (DiscardHandler) UpdateCopyOptions(opts *oras.CopyGraphOptions, fetcher content.Fetcher) {
}
