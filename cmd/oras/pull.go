package main

import (
	"context"
	"fmt"

	"github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/oras"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type pullOptions struct {
	targetRef          string
	allowedMediaTypes  []string
	allowAllMediaTypes bool
	keepOldFiles       bool
	pathTraversal      bool
	output             string
	verbose            bool

	debug    bool
	configs  []string
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
	cmd.Flags().BoolVarP(&opts.keepOldFiles, "keep-old-files", "k", false, "do not replace existing files when pulling, treat them as errors")
	cmd.Flags().BoolVarP(&opts.pathTraversal, "allow-path-traversal", "T", false, "allow storing files out of the output directory")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "output directory")
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "verbose output")

	cmd.Flags().BoolVarP(&opts.debug, "debug", "d", false, "debug mode")
	cmd.Flags().StringArrayVarP(&opts.configs, "config", "c", nil, "auth config path")
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
		opts.allowedMediaTypes = []string{content.DefaultBlobMediaType, content.DefaultBlobDirMediaType}
	}

	resolver := newResolver(opts.username, opts.password, opts.configs...)
	store := content.NewFileStore(opts.output)
	defer store.Close()
	store.DisableOverwrite = opts.keepOldFiles
	store.AllowPathTraversalOnWrite = opts.pathTraversal
	desc, files, err := oras.Pull(context.Background(), resolver, opts.targetRef, store, oras.WithAllowedMediaTypes(opts.allowedMediaTypes))
	if err != nil {
		return err
	}

	if opts.verbose {
		for _, file := range files {
			if name, ok := content.ResolveName(file); ok {
				fmt.Println(name)
			}
		}
		fmt.Println("Pulled", opts.targetRef)
		fmt.Println(desc.Digest)
	}

	return nil
}
