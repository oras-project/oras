package content

import (
	"compress/gzip"
	"fmt"
	"io"

	"github.com/containerd/containerd/content"
)

// NewGunzipWriter wrap a writer with a gunzip, so that the stream is gunzipped
func NewGunzipWriter(writer content.Writer, blocksize int) content.Writer {
	if blocksize == 0 {
		blocksize = DefaultBlocksize
	}
	return NewPassthroughWriter(writer, func(r io.Reader, w io.Writer, done chan<- error) {
		gr, err := gzip.NewReader(r)
		if err != nil {
			done <- fmt.Errorf("error creating gzip reader: %v", err)
			return
		}
		// write out the uncompressed data
		b := make([]byte, blocksize, blocksize)
		for {
			var n int
			n, err = gr.Read(b)
			if err != nil && err != io.EOF {
				err = fmt.Errorf("GunzipWriter data read error: %v\n", err)
				break
			}
			l := n
			if n > len(b) {
				l = len(b)
			}
			if _, err2 := w.Write(b[:l]); err2 != nil {
				err = fmt.Errorf("GunzipWriter: error writing to underlying writer: %v", err2)
				break
			}
			if err == io.EOF {
				// clear the error
				err = nil
				break
			}
		}
		gr.Close()
		done <- err
	})
}
