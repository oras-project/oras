package main

import (
	"context"

	artifactspec "github.com/oras-project/artifacts-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras/cmd/oras/internal/option"
)

type attachOptions struct {
	option.Common
	option.Remote

	targetRef    string
	artifactType string
	fileRefs     []string
}

func attachCmd() *cobra.Command {
	var opts attachOptions
	cmd := &cobra.Command{
		Use:   "attach name[:tag|@digest] file[:type] [file...]",
		Short: "Attach files to an existed manifest",
		Long: `Attach files to an existed manifest

Example - Attach file "hi.txt" with custom artifact type "sig/example" to localhost:5000/hello:test
  oras attach localhost:5000/hello:test hi.txt --artifact-type "sig/example"
`,
		Args: cobra.MinimumNArgs(2),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.ReadPassword()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			opts.fileRefs = args[1:]
			return runAttach(opts)
		},
	}

	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func runAttach(opts attachOptions) error {
	ctx, _ := opts.SetLoggerLevel()

}

func packManifest(ctx context.Context, store *file.Store, opts *attachOptions) (desc artifactspec.Descriptor, err error)
