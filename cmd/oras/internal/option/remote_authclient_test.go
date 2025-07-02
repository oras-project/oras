package option

import (
	"crypto/tls"
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"oras.land/oras-go/v2/registry/remote/auth"
)

func TestAuthClient_UsesSimpleContextCache(t *testing.T) {
	remo := &Remote{
		UseSimpleAuth: true,
		headers:       http.Header{},
		tlsConfigFn: func() (*tls.Config, error) {
			return &tls.Config{}, nil
		},
	}

	client, err := remo.authClient("example.com", false)
	require.NoError(t, err, "authClient should not return error when UseSimpleAuth is true")

	expectedType := reflect.TypeOf(auth.NewSingleContextCache())
	actualType := reflect.TypeOf(client.Cache)
	require.Equal(t, expectedType, actualType, "expected auth.NewSingleContextCache() to be used")
}

func TestAuthClient_UsesDefaultCache(t *testing.T) {
	remo := &Remote{
		UseSimpleAuth: false,
		headers:       http.Header{},
		tlsConfigFn: func() (*tls.Config, error) {
			return &tls.Config{}, nil
		},
	}

	client, err := remo.authClient("example.com", false)
	require.NoError(t, err, "authClient should not return error when UseSimpleAuth is false")

	expectedType := reflect.TypeOf(auth.NewCache())
	actualType := reflect.TypeOf(client.Cache)
	require.Equal(t, expectedType, actualType, "expected auth.NewCache() to be used")
}
