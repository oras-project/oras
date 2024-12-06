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
	"mime"
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

// payloadSizeLimit limits the maximum size of the response body to be printed.
const payloadSizeLimit int64 = 16 * 1024 // 16 KiB

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

	// log the request
	e.Debugf("--> Request #%d\n> Request URL: %q\n> Request method: %q\n> Request headers:\n%s",
		id, req.URL, req.Method, logHeader(req.Header))

	// log the response
	resp, err = t.RoundTripper.RoundTrip(req)
	if err != nil {
		e.Errorf("<-- Response #%d\nError in getting response: %v", id, err)
	} else if resp == nil {
		e.Errorf("<-- Response #%d\nNo response obtained for request %s %q", id, req.Method, req.URL)
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

// logResponseBody prints out the response body if it is printable and within
// the size limit.
func logResponseBody(resp *http.Response) string {
	if resp.Body == nil || resp.Body == http.NoBody {
		return "   No response body to print"
	}

	// non-applicable body is not printed and remains untouched for subsequent processing
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		return "   Response body without a content type is not printed"
	}
	if !isPrintableContentType(contentType) {
		return fmt.Sprintf("   Response body of content type %q is not printed", contentType)
	}

	buf := bytes.NewBuffer(nil)
	body := resp.Body
	// restore the body by concatenating the read body with the remaining body
	resp.Body = struct {
		io.Reader
		io.Closer
	}{
		Reader: io.MultiReader(buf, body),
		Closer: body,
	}
	// read the body up to limit+1 to check if the body exceeds the limit
	if _, err := io.CopyN(buf, body, payloadSizeLimit+1); err != nil && err != io.EOF {
		return fmt.Sprintf("   Error reading response body: %v", err)
	}

	readBody := buf.String()
	if len(readBody) == 0 {
		return "   Response body is empty"
	}
	if containsCredentials(readBody) {
		return "   Response body redacted due to potential credentials"
	}
	if len(readBody) > int(payloadSizeLimit) {
		return readBody[:payloadSizeLimit] + "\n...(truncated)"
	}
	return readBody
}

// isPrintableContentType returns true if the content of contentType is printable.
func isPrintableContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}

	switch mediaType {
	case "application/json", // JSON types
		"text/plain", "text/html": // text types
		return true
	}
	return strings.HasSuffix(mediaType, "+json")
}

// containsCredentials returns true if the body contains potential credentials.
func containsCredentials(body string) bool {
	return strings.Contains(body, `"token"`) || strings.Contains(body, `"access_token"`)
}
