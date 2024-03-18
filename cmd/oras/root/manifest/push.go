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
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	digest "github.com/opencontainers/go-digest"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras/cmd/oras/internal/argument"
	"oras.land/oras/cmd/oras/internal/display/status"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/manifest"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/file"
)

type pushOptions struct {
	option.Common
	option.Descriptor
	option.Pretty
	option.Target

	concurrency int
	extraRefs   []string
	fileRef     string
	mediaType   string
}

func pushCmd() *cobra.Command {
	var opts pushOptions
	cmd := &cobra.Command{
		Use:   "push [flags] <name>[:<tag>[,<tag>][...]|@<digest>] <file>",
		Short: "Push a manifest to a registry or an OCI image layout",
		Long: `Push a manifest to a registry or an OCI image layout

Example - Push a manifest to repository 'localhost:5000/hello' and tag with 'v1':
  oras manifest push localhost:5000/hello:v1 manifest.json

Example - Push a manifest using a specific method for the Referrers API:
  oras manifest push --distribution-spec v1.1-referrers-api localhost:5000/hello:v1 manifest.json
  oras manifest push --distribution-spec v1.1-referrers-tag localhost:5000/hello:v1 manifest.json

Example - Push a manifest with content read from stdin:
  oras manifest push localhost:5000/hello:v1 -

Example - Push a manifest and output its descriptor:
  oras manifest push --descriptor localhost:5000/hello:v1 manifest.json

Example - Push a manifest to repository 'localhost:5000/hello' and output the prettified descriptor:
  oras manifest push --descriptor --pretty localhost:5000/hello manifest.json

Example - Push a manifest with specified media type to repository 'localhost:5000/hello' and tag with 'v1':
  oras manifest push --media-type application/vnd.cncf.oras.artifact.manifest.v1+json localhost:5000/hello:v1 oras_manifest.json

Example - Push a manifest to repository 'localhost:5000/hello' and tag with 'tag1', 'tag2', 'tag3':
  oras manifest push localhost:5000/hello:tag1,tag2,tag3 manifest.json

Example - Push a manifest to repository 'localhost:5000/hello' and tag with 'tag1', 'tag2', 'tag3' and concurrency level tuned:
  oras manifest push --concurrency 6 localhost:5000/hello:tag1,tag2,tag3 manifest.json

Example - Push a manifest to an OCI image layout folder 'layout-dir' and tag with 'v1':
  oras manifest push --oci-layout layout-dir:v1 manifest.json
`,
		Args: oerrors.CheckArgs(argument.Exactly(2), "the destination to push to and the file to read manifest content from"),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.fileRef = args[1]
			if opts.fileRef == "-" && opts.PasswordFromStdin {
				return errors.New("`-` read file from input and `--password-stdin` read password from input cannot be both used")
			}
			refs := strings.Split(args[0], ",")
			opts.RawReference = refs[0]
			opts.extraRefs = refs[1:]
			return option.Parse(&opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return pushManifest(cmd, opts)
		},
	}

	opts.EnableDistributionSpecFlag()
	option.ApplyFlags(&opts, cmd.Flags())
	cmd.Flags().StringVarP(&opts.mediaType, "media-type", "", "", "media type of manifest")
	cmd.Flags().IntVarP(&opts.concurrency, "concurrency", "", 5, "concurrency level")
	return oerrors.Command(cmd, &opts.Target)
}

func pushManifest(cmd *cobra.Command, opts pushOptions) error {
	ctx, logger := opts.WithContext(cmd.Context())
	var target oras.Target
	var err error
	target, err = opts.NewTarget(opts.Common, logger)
	if err != nil {
		return err
	}
	if repo, ok := target.(*remote.Repository); ok {
		target = repo.Manifests()
	}

	// prepare manifest content
	contentBytes, err := file.PrepareManifestContent(opts.fileRef)
	if err != nil {
		return err
	}

	// get manifest media type
	mediaType := opts.mediaType
	if opts.mediaType == "" {
		mediaType, err = manifest.ExtractMediaType(contentBytes)
		if err != nil {
			if errors.Is(err, manifest.ErrMediaTypeNotFound) {
				return &oerrors.Error{
					Err:            fmt.Errorf(`%w via the flag "--media-type" nor in %q`, err, opts.fileRef),
					Usage:          fmt.Sprintf("%s %s", cmd.Parent().CommandPath(), cmd.Use),
					Recommendation: `Please specify a valid media type in the manifest JSON or via the "--media-type" flag`,
				}
			}
			return err
		}
	}

	// prepare manifest descriptor
	desc := content.NewDescriptorFromBytes(mediaType, contentBytes)

	ref := opts.Reference
	if ref == "" {
		ref = desc.Digest.String()
	}
	match, err := matchDigest(ctx, target, ref, desc.Digest)
	if err != nil {
		return err
	}
	verbose := opts.Verbose && !opts.OutputDescriptor
	if match {
		if err := status.PrintStatus(desc, "Exists", verbose); err != nil {
			return err
		}
	} else {
		if err = status.PrintStatus(desc, "Uploading", verbose); err != nil {
			return err
		}
		if _, err := oras.TagBytes(ctx, target, mediaType, contentBytes, ref); err != nil {
			return err
		}
		if err = status.PrintStatus(desc, "Uploaded ", verbose); err != nil {
			return err
		}
	}

	tagBytesNOpts := oras.DefaultTagBytesNOptions
	tagBytesNOpts.Concurrency = opts.concurrency

	// outputs manifest's descriptor
	if opts.OutputDescriptor {
		if len(opts.extraRefs) != 0 {
			if _, err = oras.TagBytesN(ctx, target, mediaType, contentBytes, opts.extraRefs, tagBytesNOpts); err != nil {
				return err
			}
		}
		descJSON, err := opts.Marshal(desc)
		if err != nil {
			return err
		}
		return opts.Output(os.Stdout, descJSON)
	}
	status.Print("Pushed", opts.AnnotatedReference())
	if len(opts.extraRefs) != 0 {
		if _, err = oras.TagBytesN(ctx, status.NewTagStatusPrinter(target), mediaType, contentBytes, opts.extraRefs, tagBytesNOpts); err != nil {
			return err
		}
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Digest:", desc.Digest)

	return nil
}

// matchDigest checks whether the manifest's digest matches to it in the remote
// repository.
func matchDigest(ctx context.Context, resolver content.Resolver, reference string, digest digest.Digest) (bool, error) {
	got, err := resolver.Resolve(ctx, reference)
	if err != nil {
		if errors.Is(err, errdef.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return got.Digest == digest, nil
}
