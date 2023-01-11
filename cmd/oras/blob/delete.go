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

	"github.com/spf13/cobra"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras/cmd/oras/internal/option"
)

type deleteBlobOptions struct {
	option.Common
	option.Confirmation
	option.Descriptor
	option.Pretty
	option.Remote

	targetRef string
}

func deleteCmd() *cobra.Command {
	var opts deleteBlobOptions
	cmd := &cobra.Command{
		Use:     "delete [flags] <name>@<digest>",
		Aliases: []string{"remove", "rm"},
		Short:   "[Preview] Delete a blob from a remote registry",
		Long: `[Preview] Delete a blob from a remote registry

** This command is in preview and under development. **

Example - Delete a blob:
  oras blob delete localhost:5000/hello@sha256:9a201d228ebd966211f7d1131be19f152be428bd373a92071c71d8deaf83b3e5

Example - Delete a blob without prompting confirmation:
  oras blob delete --force localhost:5000/hello@sha256:9a201d228ebd966211f7d1131be19f152be428bd373a92071c71d8deaf83b3e5

Example - Delete a blob and print its descriptor:
  oras blob delete --descriptor --force localhost:5000/hello@sha256:9a201d228ebd966211f7d1131be19f152be428bd373a92071c71d8deaf83b3e5
  `,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.OutputDescriptor && !opts.Force {
				return errors.New("must apply --force to confirm the deletion if the descriptor is outputted")
			}
			return option.Parse(&opts)
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

	if _, err = repo.Reference.Digest(); err != nil {
		return fmt.Errorf("%s: blob reference must be of the form <name@digest>", opts.targetRef)
	}

	blobs := repo.Blobs()
	desc, err := blobs.Resolve(ctx, opts.targetRef)
	if err != nil {
		if errors.Is(err, errdef.ErrNotFound) {
			if opts.Force && !opts.OutputDescriptor {
				// ignore nonexistent
				fmt.Fprintf(os.Stderr, "Missing %s\n", opts.targetRef)
				return nil
			}
			return fmt.Errorf("%s: the specified blob does not exist", opts.targetRef)
		}
		return err
	}

	prompt := fmt.Sprintf("Are you sure you want to delete the blob %q?", desc.Digest)
	confirmed, err := opts.AskForConfirmation(os.Stdin, prompt)
	if err != nil {
		return err
	}
	if !confirmed {
		return nil
	}

	if err = blobs.Delete(ctx, desc); err != nil {
		return fmt.Errorf("failed to delete %s: %w", opts.targetRef, err)
	}

	if opts.OutputDescriptor {
		descJSON, err := opts.Marshal(desc)
		if err != nil {
			return err
		}
		return opts.Output(os.Stdout, descJSON)
	}

	fmt.Println("Deleted", opts.targetRef)

	return nil
}
