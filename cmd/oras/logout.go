package main

import (
	"context"

	auth "github.com/deislabs/oras/pkg/auth/docker"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type logoutOptions struct {
	hostname string

	debug   bool
	configs []string
}

func logoutCmd() *cobra.Command {
	var opts logoutOptions
	cmd := &cobra.Command{
		Use:   "logout registry",
		Short: "Log out from a remote registry",
		Long: `Log out from a remote registry

Example - Logout:
  oras logout localhost:5000
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.hostname = args[0]
			return runLogout(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.debug, "debug", "d", false, "debug mode")
	cmd.Flags().StringArrayVarP(&opts.configs, "config", "c", nil, "auth config path")
	return cmd
}

func runLogout(opts logoutOptions) error {
	if opts.debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	cli, err := auth.NewClient(opts.configs...)
	if err != nil {
		return err
	}

	return cli.Logout(context.Background(), opts.hostname)
}
