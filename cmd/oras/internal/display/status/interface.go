package status

import (
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
)

type PushHandler interface {
	OnFileLoading(name string) error
	OnEmptyArtifact() error
	TrackTarget(gt oras.GraphTarget) (oras.GraphTarget, error)
	UpdateCopyOptions(opts *oras.CopyGraphOptions, fetcher content.Fetcher)
}

type AttachHandler PushHandler
