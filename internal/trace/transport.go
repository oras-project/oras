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

package trace

import (
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

// Transport is an http.RoundTripper that keeps track of the in-flight
// request and add hooks to report HTTP tracing events.
type Transport struct {
	http.RoundTripper
	mu    sync.Mutex
	count uint64
}

func NewTransport(base http.RoundTripper) *Transport {
	return &Transport{base, sync.Mutex{}, 0}
}

// RoundTrip calls base roundtrip while keeping track of the current request.
func (t *Transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	atomic.AddUint64(&t.count, 1)

	ctx := req.Context()
	e := Logger(ctx)

	func() {
		t.mu.Lock()
		defer t.mu.Unlock()
		e.Debugf("Request #%d", t.count)
		e.Debugf("> Request URL: %q", req.URL)
		e.Debugf("> Request method: %q", req.Method)
		e.Debugf("> Request headers:")
		logHeader(req.Header, e)
	}()

	resp, err = t.RoundTripper.RoundTrip(req)
	if err != nil {
		e.Errorf("Error in getting response: %w", err)
	} else if resp == nil {
		e.Errorf("No response obtained for request %s %q", req.Method, req.URL)
	} else {
		func() {
			t.mu.Lock()
			defer t.mu.Unlock()
			e.Debugf("Response #%d", t.count)
			e.Debugf("< Response Status: %q", resp.Status)
			e.Debugf("< Response headers:")
			logHeader(resp.Header, e)
		}()
	}
	return resp, err
}

// logHeader prints out the provided header keys and values, with auth header
// scrubbed.
func logHeader(header http.Header, e logrus.FieldLogger) {
	if len(header) > 0 {
		for k, v := range header {
			if strings.EqualFold(k, "Authorization") {
				v = []string{"*****"}
			}
			e.Debugf("   %q: %q", k, strings.Join(v, ", "))
		}
	} else {
		e.Debugf("   Empty header")
	}
}
