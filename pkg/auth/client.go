package auth

import (
	"context"

	"github.com/containerd/containerd/remotes"
)

// Client provides authentication operations for remotes.
type Client interface {
	// Login logs in to a remote server identified by the hostname.
	Login(ctx context.Context, hostname, username, secret string) error
	// Logout logs out from a remote server identified by the hostname.
	Logout(ctx context.Context, hostname string) error
	// Resolver returns a new authenticated resolver.
	Resolver(ctx context.Context) (remotes.Resolver, error)
}
