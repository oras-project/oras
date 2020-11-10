package content

import (
	"context"
	"strings"

	ctrcontent "github.com/containerd/containerd/content"
)

// DecompressWriter store to decompress content and extract from tar, if needed, wrapping
// another store. By default, a FileStore will simply take each artifact and write it to
// a file, as a MemoryStore will do into memory. If the artifact is gzipped or tarred,
// you might want to store the actual object inside tar or gzip. Wrap your Store
// with DecompressStore, and it will check the media-type and, if relevant,
// gunzip and/or untar.
//
// For example:
//
//        fileStore := NewFileStore(rootPath)
//        decompressStore := store.NewDecompressStore(fileStore, blocksize)
//
type DecompressStore struct {
	ingester  ctrcontent.Ingester
	blocksize int
}

func NewDecompressStore(ingester ctrcontent.Ingester, blocksize int) DecompressStore {
	return DecompressStore{ingester, blocksize}
}

// Writer get a writer
func (d DecompressStore) Writer(ctx context.Context, opts ...ctrcontent.WriterOpt) (ctrcontent.Writer, error) {
	// the logic is straightforward:
	// - if there is a desc in the opts, and the mediatype is tar or tar+gzip, then pass the correct decompress writer
	// - else, pass the regular writer
	var (
		writer ctrcontent.Writer
		err    error
	)

	// we have to reprocess the opts to find the desc
	var wOpts ctrcontent.WriterOpts
	for _, opt := range opts {
		if err := opt(&wOpts); err != nil {
			return nil, err
		}
	}
	// figure out if compression and/or archive exists
	desc := wOpts.Desc
	// before we pass it down, we need to strip anything we are removing here
	// and possibly update the digest, since the store indexes things by digest
	hasGzip, hasTar, modifiedMediaType := checkCompression(desc.MediaType)
	wOpts.Desc.MediaType = modifiedMediaType
	opts = append(opts, ctrcontent.WithDescriptor(wOpts.Desc))
	writer, err = d.ingester.Writer(ctx, opts...)
	if err != nil {
		return nil, err
	}
	// determine if we pass it blocksize, only if positive
	writerOpts := []WriterOpt{}
	if d.blocksize > 0 {
		writerOpts = append(writerOpts, WithBlocksize(d.blocksize))
	}
	// figure out which writer we need
	if hasTar {
		writer = NewUntarWriter(writer, writerOpts...)
	}
	if hasGzip {
		writer = NewGunzipWriter(writer, writerOpts...)
	}
	return writer, nil
}

// checkCompression check if the mediatype uses gzip compression or tar.
// Returns if it has gzip and/or tar, as well as the base media type without
// those suffixes.
func checkCompression(mediaType string) (gzip, tar bool, mt string) {
	mt = mediaType
	gzipSuffix := "+gzip"
	tarSuffix := ".tar"
	if strings.HasSuffix(mt, gzipSuffix) {
		mt = mt[:len(mt)-len(gzipSuffix)]
		gzip = true
	}
	if strings.HasSuffix(mt, tarSuffix) {
		mt = mt[:len(mt)-len(tarSuffix)]
		tar = true
	}
	return
}
