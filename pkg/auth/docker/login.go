package docker

import "context"

// Login logs in to a docker registry identified by the hostname.
func (c *Client) Login(ctx context.Context, hostname, username, secret string) error {
	// @todo(shizh): require implementation
	return nil
}
