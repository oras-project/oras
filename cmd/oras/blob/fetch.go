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
	"os"
	"strings"

	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"

	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/cache"
	"oras.land/oras/internal/cas"
)

type fetchBlobOptions struct {
	option.Common
	option.Descriptor
	option.Pretty
	option.Remote

	cacheRoot  string
	outputPath string
	targetRef  string
}

func fetchCmd() *cobra.Command {
	var opts fetchBlobOptions
	cmd := &cobra.Command{
		Use:   "fetch <name@digest> [flags]",
		Short: "[Preview] Fetch a blob from a remote registry",
		Long: `[Preview] Fetch a blob from a remote registry
** This command is in preview and under development. **

Example - Fetch raw blob:
  oras blob fetch localhost:5000/hello@sha256:9a201d228ebd966211f7d1131be19f152be428bd373a92071c71d8deaf83b3e5

Example - Fetch the descriptor of a blob:
  oras blob fetch --descriptor localhost:5000/hello@sha256:9a201d228ebd966211f7d1131be19f152be428bd373a92071c71d8deaf83b3e5

Example - Fetch blob from the insecure registry:
  oras blob fetch localhost:5000/hello@sha256:9a201d228ebd966211f7d1131be19f152be428bd373a92071c71d8deaf83b3e5 --insecure
`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.cacheRoot = os.Getenv("ORAS_CACHE")
			return opts.ReadPassword()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return fetchBlob(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.outputPath, "output", "o", "", "output directory")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func fetchBlob(opts fetchBlobOptions) (err error) {
	ctx, _ := opts.SetLoggerLevel()

	if opts.outputPath == "" && !opts.OutputDescriptor {
		return errors.New("either `--output` or `--descriptor` must be provided")
	}

	if opts.outputPath == "-" && opts.OutputDescriptor {
		return errors.New("`--output -` cannot be used with `--descriptor` at the same time")
	}

	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}

	if repo.Reference.Reference == "" || !strings.Contains(opts.targetRef, "@") {
		return fmt.Errorf("%s: blob reference not support, expecting <name@digest>", opts.targetRef)
	}

	var src oras.Target = cas.BlobTarget(repo.Blobs())
	if opts.cacheRoot != "" {
		ociStore, err := oci.New(opts.cacheRoot)
		if err != nil {
			return err
		}
		src = cache.New(src, ociStore)
	}

	// fetch blob
	content, err := cas.FetchBlob(ctx, src, opts.targetRef)
	if err != nil {
		return err
	}

	// outputs blob content if "--output -" is used
	if opts.outputPath == "-" {
		_, err = os.Stdout.Write(content)
		return err
	}

	// save blob content into the local file if the output path is provided
	if opts.outputPath != "" {
		if err = os.WriteFile(opts.outputPath, content, 0666); err != nil {
			return err
		}
	}

	// outputs blob's descriptor if `--descriptor` is used
	if opts.OutputDescriptor {
		desc, err := cas.FetchDescriptor(ctx, src, opts.targetRef, nil)
		if err != nil {
			return err
		}
		err = opts.Output(os.Stdout, desc)
		if err != nil {
			return err
		}
	}

	return err
}
