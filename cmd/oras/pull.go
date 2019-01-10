package main

import (
	"context"
	"fmt"

	"github.com/shizhMSFT/oras/pkg/content"
	"github.com/shizhMSFT/oras/pkg/oras"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type pullOptions struct {
	targetRef          string
	allowedMediaTypes  []string
	allowAllMediaTypes bool
	overwrite          bool
	pathTraversal      bool
	output             string
	verbose            bool

	debug    bool
	username string
	password string
}

func pullCmd() *cobra.Command {
	var opts pullOptions
	cmd := &cobra.Command{
		Use:   "pull name[:tag|@digest]",
		Short: "Pull files from remote registry",
		Long: `Pull files from remote registry

Example - Pull only files with the "application/vnd.oci.image.layer.v1.tar" media type (default):
  oras pull localhost:5000/hello:latest

Example - Pull only files with the custom "application/vnd.me.hi" media type:
  oras pull localhost:5000/hello:latest -t application/vnd.me.hi

Example - Pull all files, any media type:
  oras pull localhost:5000/hello:latest -a
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return runPull(opts)
		},
	}

	cmd.Flags().StringArrayVarP(&opts.allowedMediaTypes, "media-type", "t", nil, "allowed media types to be pulled")
	cmd.Flags().BoolVarP(&opts.allowAllMediaTypes, "allow-all", "a", false, "allow all media types to be pulled")
	cmd.Flags().BoolVarP(&opts.overwrite, "force", "f", false, "allow overwrite on existing files")
	cmd.Flags().BoolVarP(&opts.pathTraversal, "allow-path-traversal", "T", false, "allow storing files out of the output directory")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "output directory")
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "verbose output")

	cmd.Flags().BoolVarP(&opts.debug, "debug", "d", false, "debug mode")
	cmd.Flags().StringVarP(&opts.username, "username", "u", "", "registry username")
	cmd.Flags().StringVarP(&opts.password, "password", "p", "", "registry password")
	return cmd
}

func runPull(opts pullOptions) error {
	if opts.debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if opts.allowAllMediaTypes {
		opts.allowedMediaTypes = nil
	} else if len(opts.allowedMediaTypes) == 0 {
		opts.allowedMediaTypes = []string{content.DefaultBlobMediaType}
	}

	resolver := newResolver(opts.username, opts.password)
	store := content.NewFileStore(opts.output)
	store.AllowOverwrite = opts.overwrite
	store.AllowPathTraversalOnWrite = opts.pathTraversal
	files, err := oras.Pull(context.Background(), resolver, opts.targetRef, store, opts.allowedMediaTypes...)
	if err != nil {
		return err
	}

	if opts.verbose {
		for _, file := range files {
			if name, ok := content.ResolveName(file); ok {
				fmt.Println(name)
			}
		}
	}

	return nil
}
