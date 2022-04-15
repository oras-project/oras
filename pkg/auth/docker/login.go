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

package docker

import (
	"context"

	ctypes "github.com/docker/cli/cli/config/types"
	"github.com/docker/docker/api/types"
	"github.com/moby/moby/registry"
	iface "oras.land/oras/pkg/auth"
)

// Login logs in to a docker registry identified by the hostname with custom options.
func (c *Client) Login(settings *iface.LoginSettings) error {
	hostname := resolveHostname(settings.Hostname)
	cred := types.AuthConfig{
		Username:      settings.Username,
		ServerAddress: hostname,
	}
	if settings.Username == "" {
		cred.IdentityToken = settings.Secret
	} else {
		cred.Password = settings.Secret
	}

	opts := registry.ServiceOptions{}

	if settings.Insecure {
		opts.InsecureRegistries = []string{hostname}
	}

	// Login to ensure valid credential
	remote, err := registry.NewService(opts)
	if err != nil {
		return err
	}
	ctx := settings.Context
	if ctx == nil {
		ctx = context.Background()
	}
	userAgent := settings.UserAgent
	if userAgent == "" {
		userAgent = "oras"
	}
	if _, token, err := remote.Auth(ctx, &cred, userAgent); err != nil {
		return err
	} else if token != "" {
		cred.Username = ""
		cred.Password = ""
		cred.IdentityToken = token
	}

	// Store credential
	return c.primaryCredentialsStore(hostname).Store(ctypes.AuthConfig(cred))

}

// resolveHostname resolves Docker specific hostnames
func resolveHostname(hostname string) string {
	switch hostname {
	case registry.IndexHostname, registry.IndexName, registry.DefaultV2Registry.Host:
		return registry.IndexServer
	}
	return hostname
}
