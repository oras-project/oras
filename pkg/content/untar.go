package content

import (
	"archive/tar"
	"fmt"
	"io"

	"github.com/containerd/containerd/content"
)

// NewUntarWriter wrap a writer with an untar, so that the stream is untarred
func NewUntarWriter(writer content.Writer, blocksize int) content.Writer {
	if blocksize == 0 {
		blocksize = DefaultBlocksize
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
			b := make([]byte, blocksize, blocksize)
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
	})
}
