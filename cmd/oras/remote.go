package main

import (
	"net"

	"oras.land/oras-go/v2/registry/remote"
)

func setPlainHTTP(repo *remote.Repository, plainHTTP bool) {
	switch host, _, _ := net.SplitHostPort(repo.Reference.Registry); host {
	case "localhost":
		repo.PlainHTTP = true
	default:
		repo.PlainHTTP = plainHTTP
	}
}
