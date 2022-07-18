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
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/moby/term"
	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/credential"
)

type loginOptions struct {
	option.Common
	option.Remote
	Hostname string
}

func loginCmd() *cobra.Command {
	var opts loginOptions
	cmd := &cobra.Command{
		Use:   "login registry",
		Short: "Log in to a remote registry",
		Long: `Log in to a remote registry

Example - Login with username and password from command line:
  oras login -u username -p password localhost:5000

Example - Login with username and password from stdin:
  oras login -u username --password-stdin localhost:5000

Example - Login with identity token from command line:
  oras login -p token localhost:5000

Example - Login with identity token from stdin:
  oras login --password-stdin localhost:5000

Example - Login with username and password by prompt:
  oras login localhost:5000

Example - Login with insecure registry from command line:
  oras login --insecure localhost:5000
`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.ReadPassword()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Hostname = args[0]
			return runLogin(opts)
		},
	}
	option.ApplyFlags(&opts, cmd.Flags())
	return cmd
}

func runLogin(opts loginOptions) (err error) {
	ctx, logger := opts.SetLoggerLevel()

	// https://github.com/oras-project/oras/issues/446
	if !opts.PlainHTTP && opts.Hostname == "https://ghcr.io" {
		opts.Hostname = "ghcr.io"
		logger.Warnf("No need to specify https header when logging into ghcr.io. This workaround should be removed in 0.14.0.")
	}

	// prompt for credential
	if opts.Password == "" {
		if opts.Username == "" {
			// prompt for username
			username, err := readLine("Username: ", false)
			if err != nil {
				return err
			}
			opts.Username = strings.TrimSpace(username)
		}
		if opts.Username == "" {
			// prompt for token
			if opts.Password, err = readLine("Token: ", true); err != nil {
				return err
			} else if opts.Password == "" {
				return errors.New("token required")
			}
		} else {
			// prompt for password
			if opts.Password, err = readLine("Password: ", true); err != nil {
				return err
			} else if opts.Password == "" {
				return errors.New("password required")
			}
		}
	}

	// Ping to ensure credential is valid
	remote, err := opts.Remote.NewRegistry(opts.Hostname, opts.Common)
	if err != nil {
		return err
	}
	if err = remote.Ping(ctx); err != nil {
		return err
	}

	// Store the validated credential
	store, err := credential.NewStore(opts.Configs...)
	if err != nil {
		return err
	}
	if err := store.Store(opts.Hostname, opts.Credential()); err != nil {
		return err
	}
	fmt.Println("Login Succeeded")
	return nil
}

func readLine(prompt string, slient bool) (string, error) {
	fmt.Print(prompt)
	if slient {
		fd := os.Stdin.Fd()
		state, err := term.SaveState(fd)
		if err != nil {
			return "", err
		}
		term.DisableEcho(fd, state)
		defer term.RestoreTerminal(fd, state)
	}

	reader := bufio.NewReader(os.Stdin)
	line, _, err := reader.ReadLine()
	if err != nil {
		return "", err
	}
	if slient {
		fmt.Println()
	}

	return string(line), nil
}
