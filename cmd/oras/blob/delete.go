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
	"strings"

	"github.com/spf13/cobra"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras/cmd/oras/internal/option"
)

type deleteBlobOptions struct {
	option.Common
	option.Remote

	targetRef string
}

func deleteCmd() *cobra.Command {
	var opts deleteBlobOptions
	cmd := &cobra.Command{
		Use:   "delete <name@digest> [flags]",
		Short: "[Preview] Delete a blob from a remote registry",
		Long: `[Preview] Delete a blob from a remote registry
** This command is in preview and under development. **

Example - Delete blob:
  oras blob delete localhost:5000/hello@sha256:9a201d228ebd966211f7d1131be19f152be428bd373a92071c71d8deaf83b3e5

Example - Delete blob from the insecure registry:
  oras blob delete localhost:5000/hello@sha256:9a201d228ebd966211f7d1131be19f152be428bd373a92071c71d8deaf83b3e5 --insecure
`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.ReadPassword()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return deleteBlob(opts)
		},
	}

	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func deleteBlob(opts deleteBlobOptions) (err error) {
	ctx, _ := opts.SetLoggerLevel()
	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}

	if repo.Reference.Reference == "" || !strings.Contains(opts.targetRef, "@") {
		return fmt.Errorf("%s: blob reference not support, expecting <name@digest>", opts.targetRef)
	}

	desc, err := repo.Blobs().Resolve(ctx, opts.targetRef)
	if err != nil {
		if errors.Is(err, errdef.ErrNotFound) {
			return fmt.Errorf("%s: the specified blob does not exist", opts.targetRef)
		}
		return err
	}
	if err = repo.Delete(ctx, desc); err != nil {
		return fmt.Errorf("failed to delete %s: %w", opts.targetRef, err)
	}

	fmt.Println("Deleted", opts.targetRef)

	return nil
}
