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
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras/cmd/oras/internal/argument"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/registryutil"
)

type deleteBlobOptions struct {
	option.Common
	option.Confirmation
	option.Descriptor
	option.Pretty
	option.Target
}

func deleteCmd() *cobra.Command {
	var opts deleteBlobOptions
	cmd := &cobra.Command{
		Use:     "delete [flags] <name>@<digest>",
		Aliases: []string{"remove", "rm"},
		Short:   "Delete a blob from a remote registry",
		Long: `Delete a blob from a remote registry

Example - Delete a blob:
  oras blob delete localhost:5000/hello@sha256:9a201d228ebd966211f7d1131be19f152be428bd373a92071c71d8deaf83b3e5

Example - Delete a blob without prompting confirmation:
  oras blob delete --force localhost:5000/hello@sha256:9a201d228ebd966211f7d1131be19f152be428bd373a92071c71d8deaf83b3e5

Example - Delete a blob and print its descriptor:
  oras blob delete --descriptor --force localhost:5000/hello@sha256:9a201d228ebd966211f7d1131be19f152be428bd373a92071c71d8deaf83b3e5
  `,
		Args: oerrors.CheckArgs(argument.Exactly(1), "the target blob to delete"),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.RawReference = args[0]
			if opts.OutputDescriptor && !opts.Force {
				return errors.New("must apply --force to confirm the deletion if the descriptor is outputted")
			}
			return option.Parse(&opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return deleteBlob(cmd, &opts)
		},
	}

	option.ApplyFlags(&opts, cmd.Flags())
	return oerrors.Command(cmd, &opts.Target)
}

func deleteBlob(cmd *cobra.Command, opts *deleteBlobOptions) (err error) {
	ctx, logger := opts.WithContext(cmd.Context())
	blobs, err := opts.NewBlobDeleter(opts.Common, logger)
	if err != nil {
		return err
	}
	if err := opts.EnsureReferenceNotEmpty(cmd, false); err != nil {
		return err
	}

	// add both pull and delete scope hints for dst repository to save potential delete-scope token requests during deleting
	outWriter := cmd.OutOrStdout()
	ctx = registryutil.WithScopeHint(ctx, blobs, auth.ActionPull, auth.ActionDelete)
	desc, err := blobs.Resolve(ctx, opts.Reference)
	if err != nil {
		if errors.Is(err, errdef.ErrNotFound) {
			if opts.Force && !opts.OutputDescriptor {
				// ignore nonexistent
				fmt.Fprintln(outWriter, "Missing", opts.RawReference)
				return nil
			}
			return fmt.Errorf("%s: the specified blob does not exist", opts.RawReference)
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
		return fmt.Errorf("failed to delete %s: %w", opts.RawReference, err)
	}

	if opts.OutputDescriptor {
		descJSON, err := opts.Marshal(desc)
		if err != nil {
			return err
		}
		return opts.Output(os.Stdout, descJSON)
	}

	fmt.Fprintln(outWriter, "Deleted", opts.AnnotatedReference())

	return nil
}
