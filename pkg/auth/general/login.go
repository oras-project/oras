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

package general

import (
	"oras.land/oras-go/v2/registry/remote"
	iface "oras.land/oras/pkg/auth"
)

// Login logs in to a registry identified by the hostname with custom options.
func (c *Client) Login(settings *iface.LoginSettings) error {
	reg, err := remote.NewRegistry(settings.Hostname)
	if err != nil {
		return err
	}
	reg.PlainHTTP = settings.PlainHTTP
	reg.Client = settings.GetAuthClient()
	// Login to ensure credential is valid
	return reg.Ping(settings.Context)
}
