package content

import (
	"context"
	"io"

	"github.com/containerd/containerd/content"
	"github.com/opencontainers/go-digest"
)

// PassthroughWriter takes an input stream and passes it through to an underlying writer,
// while providing the ability to manipulate the stream before it gets passed through
type PassthroughWriter struct {
	writer           content.Writer
	pipew            *io.PipeWriter
	digester         digest.Digester
	size             int64
	underlyingWriter *underlyingWriter
	reader           *io.PipeReader
	hash             *digest.Digest
	done             chan error
}

// NewPassthroughWriter creates a pass-through writer that allows for processing
// the content via an arbitrary function. The function should do whatever processing it
// wants, reading from the Reader to the Writer. When done, it must indicate via
// sending an error or nil to the Done
func NewPassthroughWriter(writer content.Writer, f func(r io.Reader, w io.Writer, done chan<- error), opts ...WriterOpt) content.Writer {
	// process opts for default
	wOpts := DefaultWriterOpts()
	for _, opt := range opts {
		if err := opt(&wOpts); err != nil {
			return nil
		}
	}

	r, w := io.Pipe()
	pw := &PassthroughWriter{
		writer:   writer,
		pipew:    w,
		digester: digest.Canonical.Digester(),
		underlyingWriter: &underlyingWriter{
			writer:   writer,
			digester: digest.Canonical.Digester(),
			hash:     wOpts.OutputHash,
		},
		reader: r,
		hash:   wOpts.InputHash,
		done:   make(chan error, 1),
	}
	go f(r, pw.underlyingWriter, pw.done)
	return pw
}

func (pw *PassthroughWriter) Write(p []byte) (n int, err error) {
	n, err = pw.pipew.Write(p)
	if pw.hash == nil {
		pw.digester.Hash().Write(p[:n])
	}
	pw.size += int64(n)
	return
}

func (pw *PassthroughWriter) Close() error {
	pw.pipew.Close()
	pw.writer.Close()
	return nil
}

// Digest may return empty digest or panics until committed.
func (pw *PassthroughWriter) Digest() digest.Digest {
	if pw.hash != nil {
		return *pw.hash
	}
	return pw.digester.Digest()
}

// Commit commits the blob (but no roll-back is guaranteed on an error).
// size and expected can be zero-value when unknown.
// Commit always closes the writer, even on error.
// ErrAlreadyExists aborts the writer.
func (pw *PassthroughWriter) Commit(ctx context.Context, size int64, expected digest.Digest, opts ...content.Opt) error {
	pw.pipew.Close()
	err := <-pw.done
	pw.reader.Close()
	if err != nil && err != io.EOF {
		return err
	}

	// Some underlying writers will validate an expected digest, so we need the option to pass it
	// that digest. That is why we caluclate the digest of the underlying writer throughout the write process.
	return pw.writer.Commit(ctx, pw.underlyingWriter.size, pw.underlyingWriter.Digest(), opts...)
}

// Status returns the current state of write
func (pw *PassthroughWriter) Status() (content.Status, error) {
	return pw.writer.Status()
}

// Truncate updates the size of the target blob
func (pw *PassthroughWriter) Truncate(size int64) error {
	return pw.writer.Truncate(size)
}

// underlyingWriter implementation of io.Writer to write to the underlying
// io.Writer
type underlyingWriter struct {
	writer   content.Writer
	digester digest.Digester
	size     int64
	hash     *digest.Digest
}

// Write write to the underlying writer
func (u *underlyingWriter) Write(p []byte) (int, error) {
	n, err := u.writer.Write(p)
	if err != nil {
		return 0, err
	}

	if u.hash == nil {
		u.digester.Hash().Write(p)
	}
	u.size += int64(len(p))
	return n, nil
}

// Size get total size written
func (u *underlyingWriter) Size() int64 {
	return u.size
}

// Digest may return empty digest or panics until committed.
func (u *underlyingWriter) Digest() digest.Digest {
	if u.hash != nil {
		return *u.hash
	}
	return u.digester.Digest()
}
