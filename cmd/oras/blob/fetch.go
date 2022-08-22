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
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"

	"oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/cache"
	"oras.land/oras/internal/cas"
)

type fetchBlobOptions struct {
	option.Common
	option.Remote

	cacheRoot       string
	fetchDescriptor bool
	output          string
	targetRef       string
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
			if !strings.Contains(opts.targetRef, "@") {
				return fmt.Errorf("%s: image reference not support, expecting <name@digest>", opts.targetRef)
			}
			return fetchBlob(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "output directory")
	cmd.Flags().BoolVarP(&opts.fetchDescriptor, "descriptor", "", false, "fetch a descriptor of the manifest")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func fetchBlob(opts fetchBlobOptions) (err error) {
	ctx, _ := opts.SetLoggerLevel()
	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}

	if repo.Reference.Reference == "" {
		return errors.NewErrInvalidReference(repo.Reference)
	}

	var src oras.Target = cas.BlobTarget{
		BlobStore: repo.Blobs(),
	}
	if opts.cacheRoot != "" {
		ociStore, err := oci.New(opts.cacheRoot)
		if err != nil {
			return err
		}
		src = cache.New(src, ociStore)
	}

	// Fetch and output
	var content []byte
	if opts.fetchDescriptor {
		content, err = cas.FetchDescriptor(ctx, src, opts.targetRef, nil)
	} else {
		content, err = cas.FetchBlob(ctx, src, opts.targetRef)
	}
	if err != nil {
		return err
	}

	if opts.output != "" {
		if err = os.WriteFile(opts.output, content, 0666); err != nil {
			return err
		}
	}

	printable := utf8.Valid(content)
	if printable {
		_, err = os.Stdout.Write(content)
	} else {
		fmt.Println("Warning: This blob is an unreadable binary file that can mess up your terminal. You can add the flag \"--output\" to save it locally.")
	}

	return err
}
