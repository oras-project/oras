package main

import (
	"context"
	"io/ioutil"

	"github.com/shizhMSFT/oras/pkg/oras"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type pushOptions struct {
	targetRef string
	filenames []string

	debug    bool
	username string
	password string
}

func pushCmd() *cobra.Command {
	var opts pushOptions
	cmd := &cobra.Command{
		Use:   "push name[:tag|@digest] file [file...]",
		Short: "Push files to remote registry",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			opts.filenames = args[1:]
			return runPush(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.debug, "debug", "d", false, "debug mode")
	cmd.Flags().StringVarP(&opts.username, "username", "u", "", "registry username")
	cmd.Flags().StringVarP(&opts.password, "password", "p", "", "registry password")
	return cmd
}

func runPush(opts pushOptions) error {
	if opts.debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	resolver := newResolver(opts.username, opts.password)

	contents := make(map[string][]byte)
	for _, filename := range opts.filenames {
		content, err := ioutil.ReadFile(filename)
		if err != nil {
			return err
		}
		contents[filename] = content
	}

	return oras.Push(context.Background(), resolver, opts.targetRef, contents)
}
