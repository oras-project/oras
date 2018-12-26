package main

import (
	"context"
	"io/ioutil"
	"path"

	"github.com/shizhMSFT/oras/pkg/oras"

	"github.com/spf13/cobra"
)

type pullOptions struct {
	targetRef string
	output    string
	username  string
	password  string
}

func pullCmd() *cobra.Command {
	var opts pullOptions
	cmd := &cobra.Command{
		Use:   "pull [OPTIONS] NAME[:TAG|@DIGEST]",
		Short: "Pull files from remote registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetRef = args[0]
			return runPull(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "output directory")
	cmd.Flags().StringVarP(&opts.username, "username", "u", "", "registry username")
	cmd.Flags().StringVarP(&opts.password, "password", "p", "", "registry password")
	return cmd
}

func runPull(opts pullOptions) error {
	resolver := newResolver(opts.username, opts.password)
	contents, err := oras.Pull(context.Background(), resolver, opts.targetRef)
	if err != nil {
		return err
	}

	for name, content := range contents {
		if opts.output != "" {
			name = path.Join(opts.output, name)
		}
		if err := ioutil.WriteFile(name, content, 0644); err != nil {
			return err
		}
	}

	return nil
}
