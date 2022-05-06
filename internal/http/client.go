package http

import (
	"context"
	"crypto/tls"
	"net/http"

	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras/internal/credential"
	"oras.land/oras/internal/trace"
	"oras.land/oras/internal/version"
)

// client option struct
type ClientOptions struct {
	Credential      auth.Credential
	CredentialStore *credential.Store
	SkipTLSVerify   bool
	Debug           bool
}

func NewClient(opts ClientOptions) remote.Client {
	client := &auth.Client{
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: opts.SkipTLSVerify,
				},
			},
		},
	}
	client.SetUserAgent("oras/" + version.GetVersion())
	if opts.Debug {
		client.Client.Transport = trace.NewTransport(client.Client.Transport)
	}

	if opts.Credential != auth.EmptyCredential {
		client.Credential = func(ctx context.Context, s string) (auth.Credential, error) {
			return opts.Credential, nil
		}
	} else if opts.CredentialStore != nil {
		client.Credential = opts.CredentialStore.Credential
	}
	return client
}
