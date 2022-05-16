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

	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras/internal/credential"
	"oras.land/oras/internal/http"
	"oras.land/oras/internal/option"
	"oras.land/oras/internal/trace"
)

type loginOptions struct {
	Hostname string
	option.Common
	option.Auth
}

func loginCmd() *cobra.Command {
	var opts loginOptions
	return &cobra.Command{
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
			opts.Hostname = args[0]
			opts.Auth.ApplyFlagsTo(cmd.Flags())
			opts.Common.ApplyFlagsTo(cmd.Flags())
			return runLogin(opts)
		},
	}
}

func runLogin(opts loginOptions) (err error) {
	var logLevel logrus.Level
	if opts.Debug {
		logLevel = logrus.DebugLevel
	} else if opts.Verbose {
		logLevel = logrus.InfoLevel
	} else {
		logLevel = logrus.WarnLevel
	}
	ctx, _ := trace.WithLoggerLevel(context.Background(), logLevel)

	// Prepare auth client
	store, err := credential.NewStore(opts.Configs...)
	if err != nil {
		return err
	}

	// Prompt credential
	if opts.FromStdin {
		password, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		opts.Password = strings.TrimSuffix(string(password), "\n")
		opts.Password = strings.TrimSuffix(opts.Password, "\r")
	} else if opts.Password == "" {
		if opts.Username == "" {
			username, err := readLine("Username: ", false)
			if err != nil {
				return err
			}
			opts.Username = strings.TrimSpace(username)
		}
		if opts.Username == "" {
			if opts.Password, err = readLine("Token: ", true); err != nil {
				return err
			} else if opts.Password == "" {
				return errors.New("token required")
			}
		} else {
			if opts.Password, err = readLine("Password: ", true); err != nil {
				return err
			} else if opts.Password == "" {
				return errors.New("password required")
			}
		}
	} else {
		fmt.Fprintln(os.Stderr, "WARNING! Using --password via the CLI is insecure. Use --password-stdin.")
	}

	// Ping to ensure credential is valid
	remote, err := remote.NewRegistry(opts.Hostname)
	if err != nil {
		return err
	}
	remote.PlainHTTP = opts.PlainHTTP
	cred := credential.Credential(opts.Username, opts.Password)
	rootCAs, err := http.LoadCertPool(opts.CaFilePath)
	if err != nil {
		return err
	}
	remote.Client = http.NewClient(http.ClientOptions{
		Credential:    cred,
		SkipTLSVerify: opts.Insecure,
		Debug:         opts.Debug,
		RootCAs:       rootCAs,
	})
	if err = remote.Ping(ctx); err != nil {
		return err
	}

	// Store the validated credential
	if err := store.Store(opts.Hostname, cred); err != nil {
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
