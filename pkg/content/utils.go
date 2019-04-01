package content

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

// ResolveName resolves name from descriptor
func ResolveName(desc ocispec.Descriptor) (string, bool) {
	name, ok := desc.Annotations[ocispec.AnnotationTitle]
	return name, ok
}

// TarDirectory walks the directory specified by path, and tar those files with a new
// path prefix.
func TarDirectory(root, prefix string, w io.Writer) error {
	tw := tar.NewWriter(w)
	defer tw.Close()
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, path)
		}

		// Rename path
		name, err := filepath.Rel(root, path)
		if err != nil {
			return errors.Wrap(err, path)
		}
		name = filepath.Join(prefix, name)
		name = filepath.ToSlash(name)

		// Generate header
		var link string
		mode := info.Mode()
		if mode&os.ModeSymlink != 0 {
			if link, err = os.Readlink(link); err != nil {
				return errors.Wrap(err, path)
			}
		}
		header, err := tar.FileInfoHeader(info, link)
		if err != nil {
			return errors.Wrap(err, path)
		}
		header.Name = name
		header.Uid = 0
		header.Gid = 0
		header.Uname = ""
		header.Gname = ""

		// Write file
		if err := tw.WriteHeader(header); err != nil {
			return errors.Wrap(err, "tar")
		}
		if mode.IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return errors.Wrap(err, path)
			}
			if _, err := io.Copy(tw, file); err != nil {
				return errors.Wrap(err, path)
			}
		}

		return nil
	}); err != nil {
		return err
	}
	return nil
}
