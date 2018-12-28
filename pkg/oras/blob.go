package oras

import ocispec "github.com/opencontainers/image-spec/specs-go/v1"

// DefaultBlobMediaType specifies the default blob media type
const DefaultBlobMediaType = ocispec.MediaTypeImageLayer

// Blob refers a blob with a media type
type Blob struct {
	MediaType string
	Content   []byte
}
