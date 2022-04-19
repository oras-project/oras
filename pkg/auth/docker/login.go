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
	"github.com/docker/cli/cli/config/types"
	"github.com/moby/moby/registry"
	"oras.land/oras-go/v2/registry/remote"
	iface "oras.land/oras/pkg/auth"
)

// Login logs in to a docker registry identified by the hostname with custom
// options.
func (c *Client) Login(settings *iface.LoginSettings) error {
	hostname := resolveHostname(settings.Hostname)
	cred := types.AuthConfig{
		Username:      settings.Secret,
		ServerAddress: hostname,
	}
	if settings.Username == "" {
		cred.IdentityToken = settings.Secret
	} else {
		cred.Password = settings.Secret
	}

	// Login to ensure valid credential
	remote, err := remote.NewRegistry(settings.Hostname)
	if err != nil {
		return err
	}
	remote.PlainHTTP = settings.PlainHTTP
	remote.Client = settings.GetAuthClient()
	if err = remote.Ping(settings.Context); err != nil {
		return err
	}

	// Store credential
	return c.primaryCredentialsStore(hostname).Store(cred)

}

// resolveHostname resolves Docker specific hostnames
func resolveHostname(hostname string) string {
	switch hostname {
	case registry.IndexHostname, registry.IndexName, registry.DefaultV2Registry.Host:
		return registry.IndexServer
	}
	return hostname
}
