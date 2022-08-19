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
	"fmt"

	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/file"
	"oras.land/oras/cmd/oras/internal/option"
)

type pushOptions struct {
	option.Common
	option.Remote

	targetRef string
	fileRef   string
	mediaType string
}

func pushCmd() *cobra.Command {
	var opts pushOptions
	cmd := &cobra.Command{
		Use:   "push name[:tag|@digest] file",
		Short: "[Preview] Push a manifest to remote registry",
		Long: `[Preview] Push a manifest to remote registry
** This command is in preview and under development. **

Example - Push a manifest to repository 'locahost:5000/hello' and tag with 'latest':
  oras manifest push localhost:5000/hello:latest manifest.json

  Example - Push an ORAS artifact manifest to repository 'locahost:5000/hello' and tag with 'latest':
  oras manifest push localhost:5000/hello:latest oras_manifest.json --media-type application/vnd.cncf.oras.artifact.manifest.v1+json

  Example - Push a manifest to the insecure registry:
  oras manifest push localhost:5000/hello:latest manifest.json
`,
		Args: cobra.ExactArgs(2),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.ReadPassword()
		},
		RunE: func(_ *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			opts.fileRef = args[1]
			return pushManifest(opts)
		},
	}

	option.ApplyFlags(&opts, cmd.Flags())
	cmd.Flags().StringVarP(&opts.mediaType, "media-type", "", "", "media type of manifest")
	return cmd
}

func pushManifest(opts pushOptions) error {
	ctx, _ := opts.SetLoggerLevel()
	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}

	var mediaType string
	if opts.mediaType != "" {
		mediaType = opts.mediaType
	} else {
		mediaType, err = file.ParseMediaType(opts.fileRef)
		if err != nil {
			return err
		}
	}

	// prepare manifest content
	desc, rc, err := file.PrepareContent(opts.fileRef, mediaType)
	if err != nil {
		return err
	}
	defer rc.Close()

	exists, err := repo.Exists(ctx, desc)
	if err != nil {
		return err
	}
	if exists {
		statusPrinter := display.StatusPrinter("Exists   ", opts.Verbose)
		if err := statusPrinter(ctx, desc); err != nil {
			return err
		}
	} else {
		if err = repo.Push(ctx, desc, rc); err != nil {
			return err
		}
	}

	if tag := repo.Reference.Reference; tag != "" {
		repo.Tag(ctx, desc, tag)
	}

	fmt.Println("Pushed", opts.targetRef)
	fmt.Println("Digest:", desc.Digest)

	return nil
}
