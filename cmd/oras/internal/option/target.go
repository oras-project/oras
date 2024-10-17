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
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/errcode"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/fileref"
)

const (
	TargetTypeRemote    = "registry"
	TargetTypeOCILayout = "oci-layout"
)

// Target struct contains flags and arguments specifying one registry or image
// layout.
// Target implements oerrors.Handler interface.
type Target struct {
	Remote
	RawReference string
	Type         string
	Reference    string //contains tag or digest
	// Path contains
	//  - path to the OCI image layout target, or
	//  - registry and repository for the remote target
	Path string

	IsOCILayout bool
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
	fs.BoolVarP(&opts.IsOCILayout, flagPrefix+"oci-layout", "", false, "set "+notePrefix+"target as an OCI image layout")
	fs.StringVar(&opts.Path, flagPrefix+"oci-layout-path", "", "set the path for the "+notePrefix+"OCI image layout target")
}

// ApplyFlagsWithPrefix applies flags to a command flag set with a prefix string.
// Commonly used for non-unary remote targets.
func (opts *Target) ApplyFlagsWithPrefix(fs *pflag.FlagSet, prefix, description string) {
	opts.applyFlagsWithPrefix(fs, prefix, description)
	opts.Remote.ApplyFlagsWithPrefix(fs, prefix, description)
}

// Parse gets target options from user input.
func (opts *Target) Parse(cmd *cobra.Command) error {
	if opts.IsOCILayout && opts.Path != "" {
		return fmt.Errorf("flag %q is not supported with %q", "oci-layout-path", "oci-layout")
	}

	switch {
	case opts.IsOCILayout:
		opts.Type = TargetTypeOCILayout
		if len(opts.headerFlags) != 0 {
			return errors.New("custom header flags cannot be used on an OCI image layout target")
		}
		return opts.parseOCILayoutReference()
	case opts.Path != "":
		opts.Type = TargetTypeOCILayout
		opts.Reference = opts.RawReference
		return nil
	default:
		opts.Type = TargetTypeRemote
		if ref, err := registry.ParseReference(opts.RawReference); err != nil {
			return &oerrors.Error{
				OperationType:  oerrors.OperationTypeParseArtifactReference,
				Err:            fmt.Errorf("%q: %w", opts.RawReference, err),
				Recommendation: "Please make sure the provided reference is in the form of <registry>/<repo>[:tag|@digest]",
			}
		} else {
			opts.Reference = ref.Reference
			ref.Reference = ""
			opts.Path = ref.String()
		}
		return opts.Remote.Parse(cmd)
	}
}

// parseOCILayoutReference parses the raw in format of <path>[:<tag>|@<digest>]
func (opts *Target) parseOCILayoutReference() error {
	raw := opts.RawReference
	var path string
	var ref string
	if idx := strings.LastIndex(raw, "@"); idx != -1 {
		// `digest` found
		path = raw[:idx]
		ref = raw[idx+1:]
	} else {
		// find `tag`
		var err error
		path, ref, err = fileref.Parse(raw, "")
		if err != nil {
			return errors.Join(err, errdef.ErrInvalidReference)
		}
	}
	opts.Path = path
	opts.Reference = ref
	return nil
}

func (opts *Target) newOCIStore() (*oci.Store, error) {
	return oci.New(opts.Path)
}

func (opts *Target) newRepository(common Common, logger logrus.FieldLogger) (*remote.Repository, error) {
	return opts.NewRepository(opts.RawReference, common, logger)
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
		return opts.NewRepository(opts.RawReference, common, logger)
	}
	return nil, fmt.Errorf("unknown target type: %q", opts.Type)
}

// EnsureReferenceNotEmpty returns formalized error when the reference is empty.
func (opts *Target) EnsureReferenceNotEmpty(cmd *cobra.Command, allowTag bool) error {
	if opts.Reference == "" {
		return oerrors.NewErrEmptyTagOrDigest(opts.RawReference, cmd, allowTag)
	}
	return nil
}

// Modify handles error during cmd execution.
func (opts *Target) Modify(cmd *cobra.Command, err error) (error, bool) {
	if opts.IsOCILayout {
		return err, false
	}

	if errors.Is(err, auth.ErrBasicCredentialNotFound) {
		return opts.DecorateCredentialError(err), true
	}

	if errors.Is(err, errdef.ErrNotFound) {
		cmd.SetErrPrefix(oerrors.RegistryErrorPrefix)
		return err, true
	}

	var errResp *errcode.ErrorResponse
	if errors.As(err, &errResp) {
		ref := registry.Reference{Registry: opts.RawReference}
		if errResp.URL.Host != ref.Host() {
			// raw reference is not registry host
			var parseErr error
			ref, parseErr = registry.ParseReference(opts.RawReference)
			if parseErr != nil {
				// this should not happen
				return err, false
			}
			if errResp.URL.Host != ref.Host() {
				// not handle if the error is not from the target
				return err, false
			}
		}

		cmd.SetErrPrefix(oerrors.RegistryErrorPrefix)
		ret := &oerrors.Error{
			Err: oerrors.TrimErrResp(err, errResp),
		}

		if ref.Registry == "docker.io" && errResp.StatusCode == http.StatusUnauthorized {
			if ref.Repository != "" && !strings.Contains(ref.Repository, "/") {
				// docker.io/xxx -> docker.io/library/xxx
				ref.Repository = "library/" + ref.Repository
				ret.Recommendation = fmt.Sprintf("Namespace seems missing. Do you mean `%s %s`?", cmd.CommandPath(), ref)
			}
		}
		return ret, true
	}
	return err, false
}

// BinaryTarget struct contains flags and arguments specifying two registries or
// image layouts.
// BinaryTarget implements errors.Handler interface.
type BinaryTarget struct {
	From        Target
	To          Target
	resolveFlag []string
}

// EnsureSourceTargetReferenceNotEmpty ensures that the from target reference is not empty.
func (opts *BinaryTarget) EnsureSourceTargetReferenceNotEmpty(cmd *cobra.Command) error {
	if opts.From.Reference == "" {
		return oerrors.NewErrEmptyTagOrDigest(opts.From.RawReference, cmd, true)
	}
	return nil
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
func (opts *BinaryTarget) Parse(cmd *cobra.Command) error {
	opts.From.warned = make(map[string]*sync.Map)
	opts.To.warned = opts.From.warned
	// resolve are parsed in array order, latter will overwrite former
	opts.From.resolveFlag = append(opts.resolveFlag, opts.From.resolveFlag...)
	opts.To.resolveFlag = append(opts.resolveFlag, opts.To.resolveFlag...)
	return Parse(cmd, opts)
}

// Modify handles error during cmd execution.
func (opts *BinaryTarget) Modify(cmd *cobra.Command, err error) (error, bool) {
	if modifiedErr, modified := opts.From.Modify(cmd, err); modified {
		return modifiedErr, modified
	}
	return opts.To.Modify(cmd, err)
}
