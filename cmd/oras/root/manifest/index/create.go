package index

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras/cmd/oras/internal/command"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/contentutil"
)

type createOptions struct {
	option.Common
	option.Target

	sources []option.Target
}

func createCmd() *cobra.Command {
	var opts createOptions
	cmd := &cobra.Command{
		Use:   "create [flags] --repo <repo-reference> <name>[:<tag>|@<digest>] [...]",
		Short: "create and push an index from provided manifests",
		Long: `create and push an index to a repository or an OCI image layout
Example - create an index from source manifests tagged s1, s2, s3 in the repository
 localhost:5000/hello, and push the index without tagging it :
  oras index create --repo localhost:5000/hello s1 s2 s3
Example - create an index from source manifests tagged s1, s2, s3 in the repository
 localhost:5000/hello, and push the index with tag 'latest' :
  oras index create --repo localhost:5000/hello --tag latest s1 s2 s3
Example - create an index from source manifests using both tags and digests, 
 and push the index with tag 'latest' :
  oras index create --repo localhost:5000/hello --tag latest s1 sha256:xxx s3
`,
		Args: cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// parse the target index reference
			opts.RawReference = args[0]
			repo, _, _ := strings.Cut(opts.RawReference, ":")

			// parse the source manifests
			opts.sources = make([]option.Target, len(args)-1)
			for i, a := range args[1:] {
				var ref string
				if contentutil.IsDigest(a) {
					ref = fmt.Sprintf("%s@%s", repo, a)
				} else {
					ref = fmt.Sprintf("%s:%s", repo, a)
				}
				m := option.Target{RawReference: ref, Remote: opts.Remote}
				if err := m.Parse(cmd); err != nil {
					return err
				}
				opts.sources[i] = m
			}

			return option.Parse(cmd, &opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return createIndex(cmd, opts)
		},
	}

	option.ApplyFlags(&opts, cmd.Flags())
	return oerrors.Command(cmd, &opts.Target)
}

func createIndex(cmd *cobra.Command, opts createOptions) error {
	ctx, logger := command.GetLogger(cmd, &opts.Common)
	dst, err := opts.NewTarget(opts.Common, logger)
	if err != nil {
		return err
	}
	// we assume that the sources and the to be created index are all in the same
	// repository, so no copy is needed
	manifests, err := resolveSourceManifests(cmd, opts, logger)
	if err != nil {
		return err
	}
	desc, reader := packIndex(manifests)
	if err := pushIndex(ctx, dst, desc, opts.Reference, reader); err != nil {
		return err
	}
	return nil
}

func resolveSourceManifests(cmd *cobra.Command, destOpts createOptions, logger logrus.FieldLogger) ([]ocispec.Descriptor, error) {
	var resolved []ocispec.Descriptor
	for _, source := range destOpts.sources {
		var err error
		// prepare sourceTarget target
		sourceTarget, err := source.NewReadonlyTarget(cmd.Context(), destOpts.Common, logger)
		if err != nil {
			return []ocispec.Descriptor{}, err
		}
		if err := source.EnsureReferenceNotEmpty(cmd, false); err != nil {
			return []ocispec.Descriptor{}, err
		}
		var desc ocispec.Descriptor
		desc, err = oras.Resolve(cmd.Context(), sourceTarget, source.Reference, oras.DefaultResolveOptions)
		if err != nil {
			return []ocispec.Descriptor{}, fmt.Errorf("failed to resolve %s: %w", source.Reference, err)
		}
		resolved = append(resolved, desc)
	}
	return resolved, nil
}

func packIndex(manifests []ocispec.Descriptor) (ocispec.Descriptor, io.Reader) {
	// todo: oras-go needs PackIndex
	index := ocispec.Index{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		MediaType: ocispec.MediaTypeImageIndex,
		Manifests: manifests,
		// todo: annotations
	}
	content, _ := json.Marshal(index)
	desc := ocispec.Descriptor{
		Digest:    digest.FromBytes(content),
		MediaType: ocispec.MediaTypeImageIndex,
		Size:      int64(len(content)),
	}
	return desc, bytes.NewReader(content)
}

func pushIndex(ctx context.Context, dst oras.Target, desc ocispec.Descriptor, ref string, content io.Reader) error {
	if refPusher, ok := dst.(registry.ReferencePusher); ok {
		if ref != "" {
			return refPusher.PushReference(ctx, desc, content, ref)
		}
	}
	if err := dst.Push(ctx, desc, content); err != nil {
		w := errors.Unwrap(err)
		if w != errdef.ErrAlreadyExists {
			return err
		}
	}
	if ref == "" {
		fmt.Println("Digest of the pushed index: ", desc.Digest)
		return nil
	}
	return dst.Tag(ctx, desc, ref)
}
