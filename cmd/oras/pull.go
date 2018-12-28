package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/shizhMSFT/oras/pkg/oras"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type pullOptions struct {
	targetRef         string
	allowedMediaTypes []string
	output            string
	verbose           bool

	debug    bool
	username string
	password string
}

func pullCmd() *cobra.Command {
	var opts pullOptions
	cmd := &cobra.Command{
		Use:   "pull name[:tag|@digest]",
		Short: "Pull files from remote registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return runPull(opts)
		},
	}

	cmd.Flags().StringArrayVarP(&opts.allowedMediaTypes, "allowed-media-type", "t", nil, "allowed media types to be pulled")
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

	resolver := newResolver(opts.username, opts.password)
	blobs, err := oras.Pull(context.Background(), resolver, opts.targetRef, opts.allowedMediaTypes...)
	if err != nil {
		return err
	}

	for name, blob := range blobs {
		if opts.output != "" {
			name = path.Join(opts.output, name)
		}
		if err := ioutil.WriteFile(name, blob.Content, 0644); err != nil {
			return err
		}
		if opts.verbose {
			fmt.Println(name)
		}
	}

	return nil
}
