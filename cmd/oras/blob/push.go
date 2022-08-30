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
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/file"
)

type pushBlobOptions struct {
	option.Common
	option.Descriptor
	option.Pretty
	option.Remote

	fileRef   string
	targetRef string
}

func pushCmd() *cobra.Command {
	var opts pushBlobOptions
	cmd := &cobra.Command{
		Use:   "push <name> file [flags]",
		Short: "[Preview] Push a blob to a remote registry",
		Long: `[Preview] Push a blob to a remote registry
** This command is in preview and under development. **

Example - Push blob "hi.txt":
  oras blob push localhost:5000/hello hi.txt

Example - Push blob from stdin:
oras blob push localhost:5000/hello -

Example - Push blob "hi.txt" and output the descriptor
  oras blob push localhost:5000/hello hi.txt --descriptor

Example - Push blob "hi.txt" and output the prettified descriptor
  oras blob push localhost:5000/hello hi.txt --descriptor --pretty

Example - Push blob without TLS:
  oras blob push localhost:5000/hello hi.txt --insecure
`,
		Args: cobra.ExactArgs(2),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.ReadPassword()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			opts.fileRef = args[1]
			return pushBlob(opts)
		},
	}

	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func pushBlob(opts pushBlobOptions) (err error) {
	ctx, _ := opts.SetLoggerLevel()

	if opts.fileRef == "-" && opts.PasswordFromStdin {
		return errors.New("`-` read file from input and `--password-stdin` read password from input cannot be both used")
	}

	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}

	// prepare blob content
	desc, rc, err := file.PrepareContent(opts.fileRef, "application/octet-stream")
	if err != nil {
		return err
	}
	defer rc.Close()

	exists, err := repo.Exists(ctx, desc)
	if err != nil {
		return err
	}
	if !exists {
		if err = repo.Push(ctx, desc, rc); err != nil {
			return err
		}
	}

	// outputs blob's descriptor
	if opts.OutputDescriptor {
		bytes, err := opts.Marshal(desc)
		if err != nil {
			return err
		}
		err = opts.Output(os.Stdout, bytes)
		return err
	}

	if exists {
		if err := display.PrintStatus(desc, "Exists", opts.Verbose); err != nil {
			return err
		}
	}
	fmt.Println("Pushed", opts.targetRef)
	fmt.Println("Digest:", desc.Digest)

	return nil
}
