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
	"errors"
	"io"
	"os"

	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/internal/option"
)

type getManifestOptions struct {
	option.Common
	option.Remote

	targetRef string
	output    string
	pretty    bool
}

func fetchManifestCmd() *cobra.Command {
	var opts getManifestOptions
	cmd := &cobra.Command{
		Use:   "fetch-manifest <name:tag|name@digest>",
		Short: "[Preview] Fetch manifest of the target artifact",
		Long: `[Preview] Fetch manifest of the target artifact
** This command is in preview and under development. **

Example - Get manifest:
  oras get-manifest localhost:5000/hello:latest

Example - Get manifest and save to manifest.json:
  oras get-manifest -output manifest.json localhost:5000/hello:latest

Example - Get manifest with prettified json result:
  oras get-manifest --pretty localhost:5000/hello:latest
`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.ReadPassword()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return getManifest(opts)
		},
	}

	cmd.Flags().BoolVar(&opts.pretty, "pretty", false, "show prettified json result")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "file path to save the output")
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func getManifest(opts getManifestOptions) error {
	ctx, _ := opts.SetLoggerLevel()
	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}
	if repo.Reference.Reference == "" {
		return newErrInvalidReference(repo.Reference)
	}

	// Read and verify digest
	desc, rc, err := repo.FetchReference(ctx, opts.targetRef)
	if err != nil {
		return err
	}
	defer rc.Close()
	verifier := desc.Digest.Verifier()
	r := io.TeeReader(rc, verifier)

	manifest := make([]byte, desc.Size)
	_, err = io.ReadFull(r, manifest)
	if err != nil {
		return err
	}
	if desc.Size != int64(len(manifest)) || !verifier.Verified() {
		return errors.New("digest verification failed")
	}

	// Output
	var writer io.Writer
	if opts.output != "" {
		file, err := os.Create(opts.output)
		if err != nil {
			return err
		}
		defer file.Close()
		writer = file
	} else {
		writer = os.Stdout
	}
	var out bytes.Buffer
	if opts.pretty {
		json.Indent(&out, manifest, "", "\t")
	} else {
		out = *bytes.NewBuffer(manifest)
	}
	out.WriteTo(writer)
	return nil
}
