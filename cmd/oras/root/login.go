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
	"errors"
	"fmt"
	"os"
	"strings"

	credentials "github.com/oras-project/oras-credentials-go"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/credential"
	"oras.land/oras/internal/io"
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

Example - Log in with identity token from the command line flags:
  oras login -i token localhost:5000

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
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return option.Parse(&opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Hostname = args[0]
			return runLogin(cmd.Context(), opts)
		},
	}
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func runLogin(ctx context.Context, opts loginOptions) (err error) {
	ctx, logger := opts.WithContext(ctx)

	// Check if both '--username' and '--identity-token' are provided
	if opts.Username != "" && opts.IdentityToken != "" {
		return fmt.Errorf("both --username and --identity-token cannot be used together")
	}

	// If IdentityToken is provided, use it as the credential without a username
	if opts.IdentityToken != "" {
		opts.Username = ""                 // Set the username to empty since it's not required when using identity token
		opts.Password = opts.IdentityToken // Use the identity token as the password
	} else if opts.Password == "" {
		// If both '--password' and '--username' are not provided, prompt for credentials
		if opts.Username == "" {
			// Prompt for username
			username, err := readLine("Username: ", false)
			if err != nil {
				return err
			}
			opts.Username = strings.TrimSpace(username)
		}
		if opts.Username == "" {
			// Prompt for token
			if opts.Password, err = readLine("Token: ", true); err != nil {
				return err
			} else if opts.Password == "" {
				return errors.New("token required")
			}
		} else {
			// Prompt for password
			if opts.Password, err = readLine("Password: ", true); err != nil {
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
	fmt.Println("Login Succeeded")
	return nil
}

func readLine(prompt string, silent bool) (string, error) {
	fmt.Print(prompt)
	fd := int(os.Stdin.Fd())
	var bytes []byte
	var err error
	if silent && term.IsTerminal(fd) {
		if bytes, err = term.ReadPassword(fd); err == nil {
			_, err = fmt.Println()
		}
	} else {
		bytes, err = io.ReadLine(os.Stdin)
	}
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
