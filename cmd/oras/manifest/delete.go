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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"oras.land/oras-go/v2/errdef"
	ierrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/util"
)

type deleteOptions struct {
	option.Common
	option.Remote

	confirmed  bool
	descriptor bool
	targetRef  string
}

func deleteCmd() *cobra.Command {
	var opts deleteOptions
	cmd := &cobra.Command{
		Use:   "delete name[:tag|@digest]",
		Short: "[Preview] Delete a manifest from remote registry",
		Long: `[Preview] Delete a manifest from remote registry
** This command is in preview and under development. **

Example - Delete a manifest tagged with 'latest' from repository 'locahost:5000/hello':
  oras manifest delete localhost:5000/hello:latest

Example - Delete a manifest with digest '99e4703fbf30916f549cd6bfa9cdbab614b5392fbe64fdee971359a77073cdf9' from repository 'locahost:5000/hello':
  oras manifest delete localhost:5000/hello@sha:99e4703fbf30916f549cd6bfa9cdbab614b5392fbe64fdee971359a77073cdf9

Example - Delete a manifest without TLS:
  oras manifest delete localhost:5000/hello:latest
`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.ReadPassword()
		},
		RunE: func(_ *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return deleteManifest(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.confirmed, "yes", "y", false, "do not prompt for confirmation")
	cmd.Flags().BoolVarP(&opts.descriptor, "descriptor", "", false, "print the descriptor of the manifest")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func deleteManifest(opts deleteOptions) error {
	ctx, _ := opts.SetLoggerLevel()
	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}

	if repo.Reference.Reference == "" {
		return ierrors.NewErrInvalidReference(repo.Reference)
	}

	ref := opts.targetRef
	desc, err := repo.Resolve(ctx, ref)
	if err != nil {
		if errors.Is(err, errdef.ErrNotFound) {
			return fmt.Errorf("%s: the specified manifest does not exist", opts.targetRef)
		}
		return err
	}

	if !opts.confirmed {
		fmt.Printf("Are you sure you want to delete the artifact '%v' and all manifests that refer to it? (y/n):", desc.Digest)
		comfirmed, err := util.AskForComfirmation()
		if err != nil {
			return err
		}
		if !comfirmed {
			return nil
		}
	}

	if err = repo.Delete(ctx, desc); err != nil {
		return fmt.Errorf("failed to delete %s: %w", opts.targetRef, err)
	}

	fmt.Println("Deleted", opts.targetRef)

	if opts.descriptor {
		descriptorByte, err := json.Marshal(desc)
		if err != nil {
			return err
		}
		fmt.Println(string(descriptorByte))
	}

	return nil
}
