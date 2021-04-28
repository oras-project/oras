package docker

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/deislabs/oras/internal/version"
	ctypes "github.com/docker/cli/cli/config/types"
	"github.com/docker/docker/registry"
)

var (
	defaultUserAgent = fmt.Sprintf("oras/%s", version.Version)
)

// Resolver returns a new authenticated resolver.
func (c *Client) Resolver(_ context.Context, client *http.Client, plainHTTP bool, customUserAgent string) (remotes.Resolver, error) {
	headers, err := buildResolverHeaders(customUserAgent)
	if err != nil {
		return nil, err
	}
	return docker.NewResolver(docker.ResolverOptions{
		Credentials: c.Credential,
		Client:      client,
		PlainHTTP:   plainHTTP,
		Headers:     headers,
	}), nil
}

// Credential returns the login credential of the request host.
func (c *Client) Credential(hostname string) (string, string, error) {
	hostname = resolveHostname(hostname)
	var (
		auth ctypes.AuthConfig
		err  error
	)
	for _, cfg := range c.configs {
		auth, err = cfg.GetAuthConfig(hostname)
		if err != nil {
			// fall back to next config
			continue
		}
		if auth.IdentityToken != "" {
			return "", auth.IdentityToken, nil
		}
		if auth.Username == "" && auth.Password == "" {
			// fall back to next config
			continue
		}
		return auth.Username, auth.Password, nil
	}
	return "", "", err
}

// resolveHostname resolves Docker specific hostnames
func resolveHostname(hostname string) string {
	switch hostname {
	case registry.IndexHostname, registry.IndexName, registry.DefaultV2Registry.Host:
		return registry.IndexServer
	}
	return hostname
}

// buildResolverHeaders generates the headers for the resolver
func buildResolverHeaders(customUserAgent string) (http.Header, error) {
	headers := http.Header{}
	if customUserAgent != "" {
		// TODO: validate user agent string
		if customUserAgent == "INVALID" {
			return nil, errors.New("invalid user agent")
		}
		headers.Set("User-Agent", customUserAgent)
	} else {
		headers.Set("User-Agent", defaultUserAgent)
	}
	return headers, nil
}
