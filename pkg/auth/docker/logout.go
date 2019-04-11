package docker

import "context"

// Logout logs out from a docker registry identified by the hostname.
func (c *Client) Logout(_ context.Context, hostname string) error {
	hostname = resolveHostname(hostname)
	return c.primaryCredentialsStore(hostname).Erase(hostname)
}
