package util

import (
	"context"
	"crypto/tls"
	"net/http"

	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras/internal/trace"
	"oras.land/oras/internal/version"
)

func AuthCredential(username, password string) auth.Credential {
	if username == "" {
		return auth.Credential{RefreshToken: password}
	} else {
		return auth.Credential{Username: username, Password: password}
	}

}

func AuthClient(cred func(context.Context, string) (auth.Credential, error), skipTlsVerify bool, debug bool) *auth.Client {
	client := &auth.Client{
		Credential: cred,
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: skipTlsVerify,
				},
			},
		},
	}
	client.SetUserAgent("oras/" + version.GetVersion())
	if debug {
		client.Client.Transport = trace.NewTransport(client.Client.Transport)
	}
	return client
}
