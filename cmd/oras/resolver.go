package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	auth "github.com/oras-project/oras-go/pkg/auth/docker"

	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
)

func newResolver(username, password string, insecure bool, plainHTTP bool, configs ...string) remotes.Resolver {

	opts := docker.ResolverOptions{
		PlainHTTP: plainHTTP,
	}

	client := http.DefaultClient
	if insecure {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}
	opts.Client = client

	if username != "" || password != "" {
		opts.Credentials = func(hostName string) (string, string, error) {
			return username, password, nil
		}
		return docker.NewResolver(opts)
	}
	cli, err := auth.NewClient(configs...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: Error loading auth file: %v\n", err)
	}
	resolver, err := cli.Resolver(context.Background(), client, plainHTTP)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: Error loading resolver: %v\n", err)
		resolver = docker.NewResolver(opts)
	}
	return resolver
}
