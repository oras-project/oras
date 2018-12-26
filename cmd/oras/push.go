package main

import (
	"context"
	"io/ioutil"

	"github.com/shizhMSFT/oras/pkg/oras"

	"github.com/spf13/cobra"
)

type pushOptions struct {
	targetRef string
	filenames []string
	username  string
	password  string
}

func pushCmd() *cobra.Command {
	var opts pushOptions
	cmd := &cobra.Command{
		Use:   "push [OPTIONS] NAME[:TAG|@DIGEST] FILE [FILE...]",
		Short: "Push files to remote registry",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			opts.filenames = args[1:]
			return runPush(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.username, "username", "u", "", "registry username")
	cmd.Flags().StringVarP(&opts.password, "password", "p", "", "registry password")
	return cmd
}

func runPush(opts pushOptions) error {
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
