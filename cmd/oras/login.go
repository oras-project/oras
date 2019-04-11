package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	auth "github.com/deislabs/oras/pkg/auth/docker"

	"github.com/docker/docker/pkg/term"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type loginOptions struct {
	hostname  string
	fromStdin bool

	debug    bool
	configs  []string
	username string
	password string
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
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.hostname = args[0]
			return runLogin(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.debug, "debug", "d", false, "debug mode")
	cmd.Flags().StringArrayVarP(&opts.configs, "config", "c", nil, "auth config path")
	cmd.Flags().StringVarP(&opts.username, "username", "u", "", "registry username")
	cmd.Flags().StringVarP(&opts.password, "password", "p", "", "registry password or identity token")
	cmd.Flags().BoolVarP(&opts.fromStdin, "password-stdin", "", false, "read password or identity token from stdin")
	return cmd
}

func runLogin(opts loginOptions) error {
	if opts.debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	// Prepare auth client
	cli, err := auth.NewClient(opts.configs...)
	if err != nil {
		return err
	}

	// Prompt credential
	if opts.fromStdin {
		password, err := ioutil.ReadAll(os.Stdin)
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
	if err := cli.Login(context.Background(), opts.hostname, opts.username, opts.password); err != nil {
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
