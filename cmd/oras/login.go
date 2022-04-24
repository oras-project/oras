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
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/moby/term"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"oras.land/oras/pkg"
	"oras.land/oras/pkg/auth"
	"oras.land/oras/pkg/auth/docker"
)

type loginOptions struct {
	hostname  string
	fromStdin bool

	debug     bool
	credType  string
	configs   []string
	username  string
	password  string
	insecure  bool
	plainHttp bool
	verbose   bool
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
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.hostname = args[0]
			return runLogin(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.debug, "debug", "d", false, "debug mode")
	cmd.Flags().StringArrayVarP(&opts.configs, "config", "c", nil, "auth config path")
	cmd.Flags().StringVarP(&opts.credType, "cred-type", "t", "docker", "type of the saved credential")
	cmd.Flags().StringVarP(&opts.username, "username", "u", "", "registry username")
	cmd.Flags().StringVarP(&opts.password, "password", "p", "", "registry password or identity token")
	cmd.Flags().BoolVarP(&opts.fromStdin, "password-stdin", "", false, "read password or identity token from stdin")
	cmd.Flags().BoolVarP(&opts.insecure, "insecure", "k", false, "allow connections to SSL registry without certs")
	cmd.Flags().BoolVarP(&opts.plainHttp, "allow-plain-http", "", false, "allow insecure connections to registry without SSL")
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "verbose output")
	return cmd
}

func runLogin(opts loginOptions) (err error) {
	ctx := context.Background()
	if opts.debug {
		logrus.SetLevel(logrus.DebugLevel)
		ctx = pkg.TracedContext(ctx)
	} else if !opts.verbose {
		logger := logrus.New()
		logger.Out = io.Discard
		e := logger.WithContext(ctx)
		ctx = context.WithValue(ctx, loggerKey{}, e)
	}

	// Prepare auth client
	var cli auth.Client
	switch opts.credType {
	case "docker":
		cli, err = docker.NewClient(opts.configs...)
	default:
		return errors.New("Unsupported credential type '" + opts.credType + "'")
	}
	if err != nil {
		return err
	}

	// Prompt credential
	if opts.fromStdin {
		password, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		opts.password = strings.TrimSuffix(string(password), "\n")
		opts.password = strings.TrimSuffix(opts.password, "\r")
	} else if opts.password == "" {
		if opts.username == "" {
			username, err := readLine("Username: ", false)
			if err != nil {
				return err
			}
			opts.username = strings.TrimSpace(username)
		}
		if opts.username == "" {
			if opts.password, err = readLine("Token: ", true); err != nil {
				return err
			} else if opts.password == "" {
				return errors.New("token required")
			}
		} else {
			if opts.password, err = readLine("Password: ", true); err != nil {
				return err
			} else if opts.password == "" {
				return errors.New("password required")
			}
		}
	} else {
		fmt.Fprintln(os.Stderr, "WARNING! Using --password via the CLI is insecure. Use --password-stdin.")
	}

	// Login
	if err := cli.Login(&auth.LoginSettings{
		Context:   ctx,
		Hostname:  opts.hostname,
		Username:  opts.username,
		Secret:    opts.password,
		Insecure:  opts.insecure,
		PlainHTTP: opts.plainHttp,
		UserAgent: pkg.GetUserAgent(),
	}); err != nil {
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
