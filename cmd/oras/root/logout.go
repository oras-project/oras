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

package root

import (
	"context"

	credentials "github.com/oras-project/oras-credentials-go"
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
		Use:   "logout [flags] <registry>",
		Short: "Log out from a remote registry",
		Long: `Log out from a remote registry

Example - Logout:
  oras logout localhost:5000
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.hostname = args[0]
			return runLogout(cmd.Context(), opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.debug, "debug", "d", false, "debug mode")
	cmd.Flags().StringArrayVarP(&opts.configs, "registry-config", "", nil, "auth config path")
	return cmd
}

func runLogout(ctx context.Context, opts logoutOptions) error {
	if opts.debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	store, err := credential.NewStore(opts.configs...)
	if err != nil {
		return err
	}
	return credentials.Logout(ctx, store, opts.hostname)
}
