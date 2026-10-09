/*
Copyright The ORAS Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package manifest

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
)

// writeOutputFile writes the content produced by write to the file at path.
//
//   - If path is a symbolic link or an existing non-regular file (e.g.
//     /dev/stdout, a FIFO or a character device), the content is written
//     through it directly, since replacing it would break the link or the
//     stream.
//   - Otherwise the content is written to a temporary file in the same
//     directory which is then renamed to path, so that a failed or
//     interrupted write never leaves a partially written file behind. The
//     permission bits of an existing file are preserved; a new file gets
//     0666 filtered by the process umask.
func writeOutputFile(path string, write func(io.Writer) error) error {
	linfo, err := os.Lstat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		linfo = nil
	case err != nil:
		return err
	}

	if linfo != nil {
		direct := linfo.Mode()&fs.ModeSymlink != 0
		if !direct {
			direct = !linfo.Mode().IsRegular()
		}
		if direct {
			return writeInPlace(path, write)
		}
	}

	// the kernel applies the umask to the requested 0666 mode
	file, err := createTempFile(path)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		_ = file.Close()
		if !committed {
			_ = os.Remove(file.Name())
		}
	}()

	if err := write(file); err != nil {
		return err
	}
	if linfo != nil {
		if err := file.Chmod(linfo.Mode().Perm()); err != nil {
			return err
		}
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(file.Name(), path); err != nil {
		return err
	}
	committed = true
	return nil
}

// writeInPlace opens path for writing without replacing it and streams the
// content into it.
func writeInPlace(path string, write func(io.Writer) error) (err error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := file.Close(); err == nil {
			err = cerr
		}
	}()
	return write(file)
}

// createTempFile exclusively creates a new file next to path with mode 0666
// (before umask).
func createTempFile(path string) (*os.File, error) {
	for range 10000 {
		var b [6]byte
		if _, err := rand.Read(b[:]); err != nil {
			return nil, err
		}
		name := path + ".oras-tmp-" + hex.EncodeToString(b[:])
		file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		return file, err
	}
	return nil, errors.New("failed to create a temporary output file")
}
