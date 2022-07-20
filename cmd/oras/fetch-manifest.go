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

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/internal/option"
)

type fetchManifestOptions struct {
	option.Common
	option.Remote
	option.Platform

	targetRef string
	raw       bool
	indent    int
	mediaType string
}

func fetchManifestCmd() *cobra.Command {
	var opts fetchManifestOptions
	cmd := &cobra.Command{
		Use:   "fetch-manifest [flags] <name:tag|name@digest>",
		Short: "[Preview] Fetch manifest of the target artifact",
		Long: `[Preview] Fetch manifest of the target artifact
** This command is in preview and under development. **

Example - Get manifest:
  oras get-manifest localhost:5000/hello:latest

Example - Get manifest with specified media type:
  oras get-manifest --media-type 'application/vnd.oci.image.manifest.v1+json' localhost:5000/hello:latest

Example - Get manifest with raw json result:
  oras get-manifest --raw localhost:5000/hello:latest
`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.ReadPassword()
		},
		RunE: func(_ *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return fetchManifest(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.raw, "raw", "r", false, "output raw manifest without formatting")
	cmd.Flags().IntVarP(&opts.indent, "indent", "n", 2, "number of spaces for indentation")
	cmd.Flags().StringVarP(&opts.mediaType, "media-type", "", "", "accepted media types")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func fetchManifest(opts fetchManifestOptions) error {
	ctx, _ := opts.SetLoggerLevel()
	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}
	if repo.Reference.Reference == "" {
		return newErrInvalidReference(repo.Reference)
	}
	if opts.mediaType != "" {
		repo.ManifestMediaTypes = []string{opts.mediaType}
	}

	// Fetch and output
	manifest, err := opts.Platform.FetchManifest(ctx, repo, opts.targetRef)
	if err != nil {
		return err
	}
	var out bytes.Buffer
	if opts.raw {
		out = *bytes.NewBuffer(manifest)
	} else {
		json.Indent(&out, manifest, "", strings.Repeat(" ", opts.indent))
	}
	out.WriteTo(os.Stdout)
	return nil
}
