package content

import (
	"context"
	"io"
	"io/ioutil"

	"github.com/containerd/containerd/content"
	"github.com/opencontainers/go-digest"
)

// IoContentWriter writer that wraps an io.Writer, so the results can be streamed to
// an open io.Writer. For example, can be used to pull a layer and write it to a file, or device.
type IoContentWriter struct {
	writer   io.Writer
	digester digest.Digester
	size     int64
}

// NewIoContentWriter create a new IoContentWriter. blocksize is the size of the block to copy,
// in bytes, between the parent and child. The default, when 0, is to simply use
// whatever golang defaults to with io.Copy
func NewIoContentWriter(writer io.Writer, blocksize int) content.Writer {
	w := writer
	if w == nil {
		w = ioutil.Discard
	}
	ioc := &IoContentWriter{
		writer:   w,
		digester: digest.Canonical.Digester(),
	}
	return NewPassthroughWriter(ioc, func(r io.Reader, w io.Writer, done chan<- error) {
		// write out the data to the io writer
		var (
			err error
		)
		if blocksize == 0 {
			_, err = io.Copy(w, r)
		} else {
			b := make([]byte, blocksize, blocksize)
			_, err = io.CopyBuffer(w, r, b)
		}
		done <- err
	})
}

func (w *IoContentWriter) Write(p []byte) (n int, err error) {
	n, err = w.writer.Write(p)
	if err != nil {
		return 0, err
	}
	w.digester.Hash().Write(p[:n])
	w.size += int64(n)
	return
}

func (w *IoContentWriter) Close() error {
	return nil
}

// Digest may return empty digest or panics until committed.
func (w *IoContentWriter) Digest() digest.Digest {
	return w.digester.Digest()
}

// Commit commits the blob (but no roll-back is guaranteed on an error).
// size and expected can be zero-value when unknown.
// Commit always closes the writer, even on error.
// ErrAlreadyExists aborts the writer.
func (w *IoContentWriter) Commit(ctx context.Context, size int64, expected digest.Digest, opts ...content.Opt) error {
	return nil
}

// Status returns the current state of write
func (w *IoContentWriter) Status() (content.Status, error) {
	return content.Status{}, nil
}

// Truncate updates the size of the target blob
func (w *IoContentWriter) Truncate(size int64) error {
	return nil
}
