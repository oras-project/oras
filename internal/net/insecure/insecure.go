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
	"net/http"
	"strings"
	"sync"
)

type transport struct {
	http.RoundTripper
	setPlainHTTP func()
}

func NewTransport(base http.RoundTripper, setPlainHTTP func()) *transport {
	if setPlainHTTP == nil {
		setPlainHTTP = func() {}
	}
	var once sync.Once
	return &transport{
		RoundTripper: base,
		setPlainHTTP: func() {
			once.Do(setPlainHTTP)
		},
	}
}

// RoundTrip calls base roundtrip while keeping track of the current request.
func (t *transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	resp, err = t.RoundTripper.RoundTrip(req)
	if err != nil && req.URL.Scheme == "https" && !strings.Contains(err.Error(), "server gave HTTP response to HTTPS client") {
		// failed because of requesting https and get http response
		// retry with http
		req.URL.Scheme = "http"
		t.setPlainHTTP()
		return t.RoundTripper.RoundTrip(req)
	}
	return resp, err
}
