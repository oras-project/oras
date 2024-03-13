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
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras/cmd/oras/internal/argument"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/credential"
	orasio "oras.land/oras/internal/io"
)

type loginOptions struct {
	option.Common
	option.Remote
	Hostname string
}

func loginCmd() *cobra.Command {
	var opts loginOptions
	cmd := &cobra.Command{
		Use:   "login [flags] <registry>",
		Short: "Log in to a remote registry",
		Long: `Log in to a remote registry

Example - Log in with username and password from command line flags:
  oras login -u username -p password localhost:5000

Example - Log in with username and password from stdin:
  oras login -u username --password-stdin localhost:5000

Example - Log in with identity token from command line flags:
  oras login -p token localhost:5000

Example - Log in with identity token from stdin:
  oras login --password-stdin localhost:5000

Example - Log in with username and password in an interactive terminal:
  oras login localhost:5000

Example - Log in with username and password in an interactive terminal and no TLS check:
  oras login --insecure localhost:5000
`,
		Args: oerrors.CheckArgs(argument.Exactly(1), "the registry to log in to"),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return option.Parse(&opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Hostname = args[0]
			return runLogin(cmd, opts)
		},
	}
	option.ApplyFlags(&opts, cmd.Flags())
	return oerrors.Command(cmd, &opts.Remote)
}

func runLogin(cmd *cobra.Command, opts loginOptions) (err error) {
	ctx, logger := opts.WithContext(cmd.Context())
	outWriter := cmd.OutOrStdout()

	// prompt for credential
	if opts.Password == "" {
		if opts.Username == "" {
			// prompt for username
			username, err := readLine(outWriter, "Username: ", false)
			if err != nil {
				return err
			}
			opts.Username = strings.TrimSpace(username)
		}
		if opts.Username == "" {
			// prompt for token
			if opts.Password, err = readLine(outWriter, "Token: ", true); err != nil {
				return err
			} else if opts.Password == "" {
				return errors.New("token required")
			}
		} else {
			// prompt for password
			if opts.Password, err = readLine(outWriter, "Password: ", true); err != nil {
				return err
			} else if opts.Password == "" {
				return errors.New("password required")
			}
		}
	}

	store, err := credential.NewStore(opts.Configs...)
	if err != nil {
		return err
	}
	remote, err := opts.Remote.NewRegistry(opts.Hostname, opts.Common, logger)
	if err != nil {
		return err
	}
	if err = credentials.Login(ctx, store, remote, opts.Credential()); err != nil {
		return err
	}
	fmt.Fprintln(outWriter, "Login Succeeded")
	return nil
}

func readLine(outWriter io.Writer, prompt string, silent bool) (string, error) {
	fmt.Fprint(outWriter, prompt)
	fd := int(os.Stdin.Fd())
	var bytes []byte
	var err error
	if silent && term.IsTerminal(fd) {
		if bytes, err = term.ReadPassword(fd); err == nil {
			_, err = fmt.Fprintln(outWriter)
		}
	} else {
		bytes, err = orasio.ReadLine(os.Stdin)
	}
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
