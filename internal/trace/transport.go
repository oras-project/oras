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
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
)

var (
	// requestCount records the number of logged request-response pairs and will
	// be used as the unique id for the next pair.
	requestCount uint64

	// toScrub is a set of headers that should be scrubbed from the log.
	toScrub = []string{
		"Authorization",
		"Set-Cookie",
	}
)

// TODO: is this number reasonable? add docs
const bodySizeLimit int64 = 8 * 1024 // 8 KiB

// Transport is an http.RoundTripper that keeps track of the in-flight
// request and add hooks to report HTTP tracing events.
type Transport struct {
	http.RoundTripper
}

// NewTransport creates and returns a new instance of Transport
func NewTransport(base http.RoundTripper) *Transport {
	return &Transport{
		RoundTripper: base,
	}
}

// RoundTrip calls base roundtrip while keeping track of the current request.
func (t *Transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	id := atomic.AddUint64(&requestCount, 1) - 1
	ctx := req.Context()
	e := Logger(ctx)

	// TEST: to be removed
	e = e.WithField("host", req.Host)
	e = e.WithField("testkey", 123)

	// log the request
	e.Debugf("--> Request #%d\n> Request URL: %q\n> Request method: %q\n> Request headers:\n%s",
		id, req.URL, req.Method, logHeader(req.Header))

	// log the response
	resp, err = t.RoundTripper.RoundTrip(req)
	if err != nil {
		e.Errorf("Error in getting response: %w", err)
	} else if resp == nil {
		e.Errorf("No response obtained for request %s %q", req.Method, req.URL)
	} else {
		e.Debugf("<-- Response #%d\n< Response Status: %q\n< Response headers:\n%s\n< Response body:\n%s",
			id, resp.Status, logHeader(resp.Header), logResponseBody(resp))
	}
	return resp, err
}

// logHeader prints out the provided header keys and values, with auth header
// scrubbed.
func logHeader(header http.Header) string {
	if len(header) > 0 {
		headers := []string{}
		for k, v := range header {
			for _, h := range toScrub {
				if strings.EqualFold(k, h) {
					v = []string{"*****"}
				}
			}
			headers = append(headers, fmt.Sprintf("   %q: %q", k, strings.Join(v, ", ")))
		}
		return strings.Join(headers, "\n")
	}
	return "   Empty header"
}

// TODO: test and docs
func logResponseBody(resp *http.Response) string {
	if resp.Body == nil {
		return "   Empty body"
	}
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") && !strings.HasPrefix(contentType, "text/") {
		return "   Body is hidden due to unsupported content type"
	}

	// TODO: if content type is json, pretty print the json?
	var builder strings.Builder
	lr := io.LimitReader(resp.Body, bodySizeLimit)
	bodyBytes, err := io.ReadAll(lr)
	if err != nil {
		return fmt.Sprintf("   Error reading response body: %v", err)
	}
	builder.Write(bodyBytes)
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// TODO: add ... if body is larger than bodySizeLimit
	return builder.String()
}
