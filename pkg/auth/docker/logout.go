package docker

import "context"

// Logout logs out from a docker registry identified by the hostname.
func (c *Client) Logout(ctx context.Context, hostname string) error {
	// @todo(shizh): require implementation
	return nil
}
