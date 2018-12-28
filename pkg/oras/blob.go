package oras

// Blob refers a blob with a media type
type Blob struct {
	MediaType string
	Content   []byte
}
