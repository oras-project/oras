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
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
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
	ctx, _ = opts.WithContext(ctx)

	if opts.Password == "" {
		// prompt for credential
		reader := bufio.NewReader(os.Stdin)
		if opts.Username == "" {
			fmt.Print("username: ")
			username, err := readLine(reader)
			if err != nil {
				return err
			}
			opts.Username = strings.TrimSpace(string(username))
		}

		fd := int(os.Stdin.Fd())
		prompt := "password"
		if opts.Username == "" {
			prompt = "token"
		}
		fmt.Printf("%s: ", prompt)
		var bytes []byte
		if term.IsTerminal(fd) {
			bytes, err = term.ReadPassword(fd)
		} else {
			bytes, err = readLine(reader)
		}
		if err != nil {
			return
		}
		fmt.Println()
		if opts.Password = string(bytes); opts.Password == "" {
			return fmt.Errorf("%s required", prompt)
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
	// For a user case that login 'docker.io',
	// According the the behavior of Docker CLI,
	// credential should be added under key "https://index.docker.io/v1/"
	hostname := opts.Hostname
	if hostname == "docker.io" {
		hostname = "https://index.docker.io/v1/"
	}
	if err := store.Store(hostname, opts.Credential()); err != nil {
		return err
	}
	fmt.Println("Login Succeeded")
	return nil
}

func readLine(reader *bufio.Reader) (content []byte, err error) {
	var line []byte
	for more := true; more; {
		// read until no more
		line, more, err = reader.ReadLine()
		if err != nil {
			return nil, err
		}
		content = append(content, line...)
	}
	return
}
