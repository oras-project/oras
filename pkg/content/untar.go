package content

import (
	"archive/tar"
	"fmt"
	"io"

	"github.com/containerd/containerd/content"
)

// NewUntarWriter wrap a writer with an untar, so that the stream is untarred
//
// By default, it calculates the hash when writing. If the option `skipHash` is true,
// it will skip doing the hash. Skipping the hash is intended to be used only
// if you are confident about the validity of the data being passed to the writer,
// and wish to save on the hashing time.
func NewUntarWriter(writer content.Writer, opts ...WriterOpt) content.Writer {
	// process opts for default
	wOpts := DefaultWriterOpts()
	for _, opt := range opts {
		if err := opt(&wOpts); err != nil {
			return nil
		}
	}

	return NewPassthroughWriter(writer, func(r io.Reader, w io.Writer, done chan<- error) {
		tr := tar.NewReader(r)
		var err error
		for {
			_, err := tr.Next()
			if err == io.EOF {
				// clear the error, since we do not pass an io.EOF
				err = nil
				break // End of archive
			}
			if err != nil {
				// pass the error on
				err = fmt.Errorf("UntarWriter tar file header read error: %v", err)
				break
			}
			// write out the untarred data
			// we can handle io.EOF, just go to the next file
			// any other errors should stop and get reported
			b := make([]byte, wOpts.Blocksize, wOpts.Blocksize)
			for {
				var n int
				n, err = tr.Read(b)
				if err != nil && err != io.EOF {
					err = fmt.Errorf("UntarWriter file data read error: %v\n", err)
					break
				}
				l := n
				if n > len(b) {
					l = len(b)
				}
				if _, err2 := w.Write(b[:l]); err2 != nil {
					err = fmt.Errorf("UntarWriter error writing to underlying writer: %v", err2)
					break
				}
				if err == io.EOF {
					// go to the next file
					break
				}
			}
			// did we break with a non-nil and non-EOF error?
			if err != nil && err != io.EOF {
				break
			}
		}
		done <- err
	}, opts...)
}
