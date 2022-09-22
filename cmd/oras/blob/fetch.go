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

package blob

import (
	"errors"
	"fmt"
	"io"
	"os"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras/cmd/oras/internal/option"
)

type fetchBlobOptions struct {
	option.Cache
	option.Common
	option.Descriptor
	option.Pretty
	option.Remote

	outputPath string
	targetRef  string
}

func fetchCmd() *cobra.Command {
	var opts fetchBlobOptions
	cmd := &cobra.Command{
		Use:   "fetch [flags] {--output <file>|--descriptor} <name>@<digest>",
		Short: "[Preview] Fetch a blob from a remote registry",
		Long: `[Preview] Fetch a blob from a remote registry

** This command is in preview and under development. **

Example - Fetch the blob and save it to a local file:
  oras blob fetch --output blob.tar.gz localhost:5000/hello@sha256:9a201d228ebd966211f7d1131be19f152be428bd373a92071c71d8deaf83b3e5

Example - Fetch the blob and stdout the raw blob content:
  oras blob fetch --output - localhost:5000/hello@sha256:9a201d228ebd966211f7d1131be19f152be428bd373a92071c71d8deaf83b3e5

Example - Fetch and stdout the descriptor of a blob:
  oras blob fetch --descriptor localhost:5000/hello@sha256:9a201d228ebd966211f7d1131be19f152be428bd373a92071c71d8deaf83b3e5

Example - Fetch the blob, save it to a local file and stdout the descriptor:
  oras blob fetch --output blob.tar.gz --descriptor localhost:5000/hello@sha256:9a201d228ebd966211f7d1131be19f152be428bd373a92071c71d8deaf83b3e5
`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.outputPath == "" && !opts.OutputDescriptor {
				return errors.New("either `--output` or `--descriptor` must be provided")
			}

			if opts.outputPath == "-" && opts.OutputDescriptor {
				return errors.New("`--output -` cannot be used with `--descriptor` at the same time")
			}

			return opts.ReadPassword()
		},
		Aliases: []string{"get"},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return fetchBlob(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.outputPath, "output", "o", "", "output file path")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func fetchBlob(opts fetchBlobOptions) (fetchErr error) {
	ctx, _ := opts.SetLoggerLevel()

	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}

	if _, err = repo.Reference.Digest(); err != nil {
		return fmt.Errorf("%s: blob reference must be of the form <name@digest>", opts.targetRef)
	}

	src, err := opts.CachedTarget(repo.Blobs())
	if err != nil {
		return err
	}

	var desc ocispec.Descriptor
	if opts.outputPath == "" {
		// fetch blob descriptor only
		desc, err = oras.Resolve(ctx, src, opts.targetRef, oras.DefaultResolveOptions)
		if err != nil {
			return err
		}
	} else {
		// fetch blob content
		var rc io.ReadCloser
		desc, rc, err = oras.Fetch(ctx, src, opts.targetRef, oras.DefaultFetchOptions)
		if err != nil {
			return err
		}
		defer rc.Close()
		vr := content.NewVerifyReader(rc, desc)

		// outputs blob content if "--output -" is used
		if opts.outputPath == "-" {
			if _, err := io.Copy(os.Stdout, vr); err != nil {
				return err
			}
			return vr.Verify()
		}

		// save blob content into the local file if the output path is provided
		file, err := os.Create(opts.outputPath)
		if err != nil {
			return err
		}
		defer func() {
			if err := file.Close(); fetchErr == nil {
				fetchErr = err
			}
		}()

		if _, err := io.Copy(file, vr); err != nil {
			return err
		}
		if err := vr.Verify(); err != nil {
			return err
		}
	}

	// outputs blob's descriptor if `--descriptor` is used
	if opts.OutputDescriptor {
		descJSON, err := opts.Marshal(desc)
		if err != nil {
			return err
		}
		if err := opts.Output(os.Stdout, descJSON); err != nil {
			return err
		}
	}

	return nil
}
