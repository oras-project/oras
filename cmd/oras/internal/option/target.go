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

package option

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/fileref"
)

const (
	TargetTypeRemote    = "registry"
	TargetTypeOCILayout = "oci-layout"
)

// Target struct contains flags and arguments specifying one registry or image
// layout.
type Target struct {
	Remote
	RawReference string
	Type         string
	Reference    string //contains tag or digest
	// Path contains
	//  - path to the OCI image layout target, or
	//  - registry and repository for the remote target
	Path string

	isOCILayout bool
}

// ApplyFlags applies flags to a command flag set for unary target
func (opts *Target) ApplyFlags(fs *pflag.FlagSet) {
	opts.applyFlagsWithPrefix(fs, "", "")
	opts.Remote.ApplyFlags(fs)
}

// AnnotatedReference returns full printable reference.
func (opts *Target) AnnotatedReference() string {
	return fmt.Sprintf("[%s] %s", opts.Type, opts.RawReference)
}

// applyFlagsWithPrefix applies flags to fs with prefix and description.
// The complete form of the `target` flag is designed to be
//
//	--target type=<type>[[,<key>=<value>][...]]
//
// For better UX, the boolean flag `--oci-layout` is introduced as an alias of
// `--target type=oci-layout`.
// Since there is only one target type besides the default `registry` type,
// the full form is not implemented until a new type comes in.
func (opts *Target) applyFlagsWithPrefix(fs *pflag.FlagSet, prefix, description string) {
	flagPrefix, notePrefix := applyPrefix(prefix, description)
	fs.BoolVarP(&opts.isOCILayout, flagPrefix+"oci-layout", "", false, "set "+notePrefix+"target as an OCI image layout")
}

// ApplyFlagsWithPrefix applies flags to a command flag set with a prefix string.
// Commonly used for non-unary remote targets.
func (opts *Target) ApplyFlagsWithPrefix(fs *pflag.FlagSet, prefix, description string) {
	opts.applyFlagsWithPrefix(fs, prefix, description)
	opts.Remote.ApplyFlagsWithPrefix(fs, prefix, description)
}

// Parse gets target options from user input.
func (opts *Target) Parse() error {
	switch {
	case opts.isOCILayout:
		opts.Type = TargetTypeOCILayout
		if len(opts.headerFlags) != 0 {
			return errors.New("custom header flags cannot be used on an OCI image layout target")
		}
		return nil
	default:
		opts.Type = TargetTypeRemote
		return opts.Remote.Parse()
	}
}

// parseOCILayoutReference parses the raw in format of <path>[:<tag>|@<digest>]
func parseOCILayoutReference(raw string) (path string, ref string, err error) {
	if idx := strings.LastIndex(raw, "@"); idx != -1 {
		// `digest` found
		path = raw[:idx]
		ref = raw[idx+1:]
	} else {
		// find `tag`
		path, ref, err = fileref.Parse(raw, "")
	}
	return
}

func (opts *Target) newOCIStore() (*oci.Store, error) {
	var err error
	opts.Path, opts.Reference, err = parseOCILayoutReference(opts.RawReference)
	if err != nil {
		return nil, err
	}
	return oci.New(opts.Path)
}

func (opts *Target) newRepository(common Common, logger logrus.FieldLogger) (*remote.Repository, error) {
	repo, err := opts.NewRepository(opts.RawReference, common, logger)
	if err != nil {
		return nil, err
	}
	tmp := repo.Reference
	tmp.Reference = ""
	opts.Path = tmp.String()
	opts.Reference = repo.Reference.Reference
	return repo, nil
}

// NewTarget generates a new target based on opts.
func (opts *Target) NewTarget(common Common, logger logrus.FieldLogger) (oras.GraphTarget, error) {
	switch opts.Type {
	case TargetTypeOCILayout:
		return opts.newOCIStore()
	case TargetTypeRemote:
		return opts.newRepository(common, logger)
	}
	return nil, fmt.Errorf("unknown target type: %q", opts.Type)
}

type ResolvableDeleter interface {
	content.Resolver
	content.Deleter
}

// NewBlobDeleter generates a new blob deleter based on opts.
func (opts *Target) NewBlobDeleter(common Common, logger logrus.FieldLogger) (ResolvableDeleter, error) {
	switch opts.Type {
	case TargetTypeOCILayout:
		return opts.newOCIStore()
	case TargetTypeRemote:
		repo, err := opts.newRepository(common, logger)
		if err != nil {
			return nil, err
		}
		return repo.Blobs(), nil
	}
	return nil, fmt.Errorf("unknown target type: %q", opts.Type)
}

// NewManifestDeleter generates a new blob deleter based on opts.
func (opts *Target) NewManifestDeleter(common Common, logger logrus.FieldLogger) (ResolvableDeleter, error) {
	switch opts.Type {
	case TargetTypeOCILayout:
		return opts.newOCIStore()
	case TargetTypeRemote:
		repo, err := opts.newRepository(common, logger)
		if err != nil {
			return nil, err
		}
		return repo.Manifests(), nil
	}
	return nil, fmt.Errorf("unknown target type: %q", opts.Type)
}

// ReadOnlyGraphTagFinderTarget represents a read-only graph target with tag
// finder capability.
type ReadOnlyGraphTagFinderTarget interface {
	oras.ReadOnlyGraphTarget
	registry.TagLister
}

// NewReadonlyTargets generates a new read only target based on opts.
func (opts *Target) NewReadonlyTarget(ctx context.Context, common Common, logger logrus.FieldLogger) (ReadOnlyGraphTagFinderTarget, error) {
	switch opts.Type {
	case TargetTypeOCILayout:
		var err error
		opts.Path, opts.Reference, err = parseOCILayoutReference(opts.RawReference)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(opts.Path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, fmt.Errorf("invalid argument %q: failed to find path %q: %w", opts.RawReference, opts.Path, err)
			}
			return nil, err
		}
		if info.IsDir() {
			return oci.NewFromFS(ctx, os.DirFS(opts.Path))
		}
		store, err := oci.NewFromTar(ctx, opts.Path)
		if err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) {
				return nil, fmt.Errorf("%q does not look like a tar archive: %w", opts.Path, err)
			}
			return nil, err
		}
		return store, nil
	case TargetTypeRemote:
		repo, err := opts.NewRepository(opts.RawReference, common, logger)
		if err != nil {
			return nil, err
		}
		tmp := repo.Reference
		tmp.Reference = ""
		opts.Path = tmp.String()
		opts.Reference = repo.Reference.Reference
		return repo, nil
	}
	return nil, fmt.Errorf("unknown target type: %q", opts.Type)
}

// EnsureReferenceNotEmpty ensures whether the tag or digest is empty.
func (opts *Target) EnsureReferenceNotEmpty() error {
	if opts.Reference == "" {
		return oerrors.NewErrEmptyTagOrDigestStr(opts.RawReference)
	}
	return nil
}

// BinaryTarget struct contains flags and arguments specifying two registries or
// image layouts.
type BinaryTarget struct {
	From        Target
	To          Target
	resolveFlag []string
}

// EnableDistributionSpecFlag set distribution specification flag as applicable.
func (opts *BinaryTarget) EnableDistributionSpecFlag() {
	opts.From.EnableDistributionSpecFlag()
	opts.To.EnableDistributionSpecFlag()
}

// ApplyFlags applies flags to a command flag set fs.
func (opts *BinaryTarget) ApplyFlags(fs *pflag.FlagSet) {
	opts.From.ApplyFlagsWithPrefix(fs, "from", "source")
	opts.To.ApplyFlagsWithPrefix(fs, "to", "destination")
	fs.StringArrayVarP(&opts.resolveFlag, "resolve", "", nil, "base DNS rules formatted in `host:port:address[:address_port]` for --from-resolve and --to-resolve")
}

// Parse parses user-provided flags and arguments into option struct.
func (opts *BinaryTarget) Parse() error {
	opts.From.warned = make(map[string]*sync.Map)
	opts.To.warned = opts.From.warned
	// resolve are parsed in array order, latter will overwrite former
	opts.From.resolveFlag = append(opts.resolveFlag, opts.From.resolveFlag...)
	opts.To.resolveFlag = append(opts.resolveFlag, opts.To.resolveFlag...)
	return Parse(opts)
}
