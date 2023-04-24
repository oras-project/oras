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

package insecure

import (
	"crypto/tls"
	"net/http"
	"sync/atomic"
)

type transport struct {
	base      http.RoundTripper
	host      string
	forceHTTP atomic.Bool
}

// NewTransport generates a new trasport with insecure retry on host.
func NewTransport(base http.RoundTripper, host string) *transport {
	return &transport{
		base: base,
		host: host,
	}
}

// RoundTrip wraps base roundtrip with conditional insecure retry.
func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Host != t.host {
		return t.base.RoundTrip(req)
	}
	if ok := t.forceHTTP.Load(); ok {
		req.URL.Scheme = "http"
		return t.base.RoundTrip(req)
	}
	resp, err := t.base.RoundTrip(req)
	if err != nil && req.URL.Scheme == "https" {
		if tlsErr, ok := err.(tls.RecordHeaderError); ok {
			// If we get a bad TLS record header, check to see if the
			// response looks like HTTP and give a more helpful error.
			// See golang.org/issue/11111.
			if string(tlsErr.RecordHeader[:]) == "HTTP/" {
				t.forceHTTP.Store(true)
				req.URL.Scheme = "http"
				return t.base.RoundTrip(req)
			}
		}
	}
	return resp, err
}
