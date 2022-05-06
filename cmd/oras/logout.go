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
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"oras.land/oras/internal/credential"
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

	store, err := credential.NewStore()
	if err != nil {
		return err
	}

	return store.Erase(opts.hostname)
}
