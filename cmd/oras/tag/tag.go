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

package tag

import (
	"context"
	"fmt"
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
)

type tagOptions struct {
	option.Common
	option.Remote

	defaultTagNOptions oras.TagNOptions
	srcRef             string
	targetRef          []string
}

func TagCmd() *cobra.Command {
	var opts tagOptions
	cmd := &cobra.Command{
		Use:   "tag [flags] name<:tag|@digest> <new_tag...>",
		Short: "[Preview] tag a manifest in the remote registry",
		Long: `[Preview] tag a manifest in the remote registry

** This command is in preview and under development. **

Example - Tag the manifest 'v1.0.1' in 'locahost:5000/hello' to 'v1.0.2':
  oras tag localhost:5000/hello:v1.0.1 v1.0.2

Example - Tag the manifest with digest sha256:9463e0d192846bc994279417b50114606712d516aab45f4d8b31cbc6e46aad71 to 'v1.0.2'
  oras tag localhost:5000/hello@sha256:9463e0d192846bc994279417b50114606712d516aab45f4d8b31cbc6e46aad71 v1.0.2

Example - Tag the manifest 'v1.0.1' in 'locahost:5000/hello' to 'v1.0.2' with the custom concurrency number of 1:
  oras tag --concurrency 1 localhost:5000/hello:v1.0.1 v1.0.2
`,
		Args: cobra.MinimumNArgs(2),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.ReadPassword()
		},
		RunE: func(_ *cobra.Command, args []string) error {
			opts.srcRef = args[0]
			opts.targetRef = args[1:]
			return tagManifest(opts)
		},
	}

	cmd.Flags().Int64VarP(&opts.defaultTagNOptions.Concurrency, "concurrency", "", 0, "provide concurrency number, default is 5")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func tagManifest(opts tagOptions) error {
	ctx, _ := opts.SetLoggerLevel()
	repo, err := opts.NewRepository(opts.srcRef, opts.Common)
	if err != nil {
		return err
	}

	if repo.Reference.Reference == "" {
		return errors.NewErrInvalidReference(repo.Reference)
	}

	rp := &wrapper{
		repo,
	}

	return oras.TagN(ctx, rp, opts.srcRef, opts.targetRef, opts.defaultTagNOptions)
}

type wrapper struct {
	*remote.Repository
}

// Override PushReference method to print off which tag(s) were added successfully.
func (w *wrapper) PushReference(ctx context.Context, expected ocispec.Descriptor, content io.Reader, reference string) error {
	if err := w.Repository.PushReference(ctx, expected, content, reference); err != nil {
		return fmt.Errorf("failed %s", reference)
	}
	fmt.Println("tagged", reference)
	return nil
}
