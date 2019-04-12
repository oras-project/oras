package oras

import (
	"path/filepath"
	"strings"

	orascontent "github.com/deislabs/oras/pkg/content"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

type pushOpts struct {
	config              *ocispec.Descriptor
	configAnnotations   map[string]string
	manifestAnnotations map[string]string
	validateName        func(desc ocispec.Descriptor) error
}

func pushOptsDefaults() *pushOpts {
	return &pushOpts{
		validateName: ValidateNameAsPath,
	}
}

// PushOpt allows callers to set options on the oras push
type PushOpt func(o *pushOpts) error

// WithConfig overrides the config
func WithConfig(config ocispec.Descriptor) PushOpt {
	return func(o *pushOpts) error {
		o.config = &config
		return nil
	}
}

// WithConfigAnnotations overrides the config annotations
func WithConfigAnnotations(annotations map[string]string) PushOpt {
	return func(o *pushOpts) error {
		o.configAnnotations = annotations
		return nil
	}
}

// WithManifestAnnotations overrides the manifest annotations
func WithManifestAnnotations(annotations map[string]string) PushOpt {
	return func(o *pushOpts) error {
		o.manifestAnnotations = annotations
		return nil
	}
}

// WithNameValidation validates the image title in the descriptor.
// Pass nil to disable name validation.
func WithNameValidation(validate func(desc ocispec.Descriptor) error) PushOpt {
	return func(o *pushOpts) error {
		o.validateName = validate
		return nil
	}
}

// ValidateNameAsPath validates name in the descriptor as file path in order
// to generate good packages intended to be pulled using the FileStore or
// the oras cli.
func ValidateNameAsPath(desc ocispec.Descriptor) error {
	// no empty name
	path, ok := orascontent.ResolveName(desc)
	if !ok || path == "" {
		return orascontent.ErrNoName
	}

	// path should be clean
	if target := filepath.ToSlash(filepath.Clean(path)); target != path {
		return errors.Wrap(ErrDirtyPath, path)
	}

	// path should be slash-separated
	if target := filepath.ToSlash(path); target != path {
		return errors.Wrap(ErrPathNotSlashSeparated, path)
	}

	// disallow path traversal
	if filepath.IsAbs(path) {
		return errors.Wrap(ErrAbsolutePathDisallowed, path)
	}
	if strings.HasPrefix(path, "../") {
		return errors.Wrap(ErrPathTraversalDisallowed, path)
	}

	return nil
}
