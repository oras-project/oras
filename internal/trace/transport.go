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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
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
		"Cookie",
		"Proxy-Authorization",
		"Set-Cookie",
	}
)

// payloadSizeLimit limits the maximum size of the response body to be printed.
const payloadSizeLimit int64 = 16 * 1024 // 16 KiB

const redactedValue = "REDACTED"

// Transport is an http.RoundTripper that keeps track of the in-flight
// request and add hooks to report HTTP tracing events.
type Transport struct {
	http.RoundTripper
	sensitiveHeaders []string
}

// NewTransport creates a Transport that additionally scrubs sensitiveHeaders.
func NewTransport(base http.RoundTripper, sensitiveHeaders ...string) *Transport {
	return &Transport{
		RoundTripper:     base,
		sensitiveHeaders: sensitiveHeaders,
	}
}

// RoundTrip calls base roundtrip while keeping track of the current request.
func (t *Transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	id := atomic.AddUint64(&requestCount, 1) - 1
	ctx := req.Context()
	e := Logger(ctx)

	// log the request
	e.Debugf("--> Request #%d\n> Request URL: %q\n> Request method: %q\n> Request headers:\n%s",
		id, redactURL(req.URL), req.Method, logHeader(req.Header, t.sensitiveHeaders...))

	// log the response
	resp, err = t.RoundTripper.RoundTrip(req)
	if err != nil {
		e.Errorf("<-- Response #%d\nError in getting response: %v", id, err)
	} else if resp == nil {
		e.Errorf("<-- Response #%d\nNo response obtained for request %s %q", id, req.Method, redactURL(req.URL))
	} else {
		e.Debugf("<-- Response #%d\n< Response Status: %q\n< Response headers:\n%s\n< Response body:\n%s",
			id, resp.Status, logHeader(resp.Header, t.sensitiveHeaders...), logResponseBody(resp))
	}
	return resp, err
}

// logHeader prints header keys and values with sensitive values and URL queries scrubbed.
func logHeader(header http.Header, sensitiveHeaders ...string) string {
	if len(header) > 0 {
		headers := []string{}
		for k, v := range header {
			if isSensitiveHeader(k, sensitiveHeaders) {
				v = []string{"*****"}
			}
			if strings.EqualFold(k, "Location") || strings.EqualFold(k, "Content-Location") || strings.EqualFold(k, "Referer") {
				redacted := make([]string, len(v))
				for i, value := range v {
					redacted[i] = redactURLString(value)
				}
				v = redacted
			}
			headers = append(headers, fmt.Sprintf("   %q: %q", k, strings.Join(v, ", ")))
		}
		return strings.Join(headers, "\n")
	}
	return "   Empty header"
}

func isSensitiveHeader(name string, additional []string) bool {
	for _, candidate := range toScrub {
		if strings.EqualFold(name, candidate) {
			return true
		}
	}
	for _, candidate := range additional {
		if strings.EqualFold(name, candidate) {
			return true
		}
	}
	return false
}

// logResponseBody prints out the response body if it is printable and within
// the size limit.
func logResponseBody(resp *http.Response) string {
	if resp.Body == nil || resp.Body == http.NoBody {
		return "   No response body to print"
	}
	if resp.StatusCode >= http.StatusMultipleChoices && resp.StatusCode < http.StatusBadRequest {
		return "   Response body for redirect is not printed"
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
	if _, err := io.CopyN(buf, body, payloadSizeLimit+1); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Sprintf("   Error reading response body: %v", err)
	}

	readBody := buf.String()
	if len(readBody) == 0 {
		return "   Response body is empty"
	}
	if len(readBody) > int(payloadSizeLimit) {
		return "   Response body exceeding the trace size limit is not printed"
	}
	if containsCredentials(readBody) {
		return "   Response body redacted due to potential credentials"
	}
	return readBody
}

func redactURLString(value string) string {
	u, err := url.Parse(value)
	if err != nil {
		return redactedValue
	}
	return redactURL(u)
}

func redactURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	redacted := *u
	if redacted.User != nil {
		redacted.User = url.User(redactedValue)
	}
	query := redacted.Query()
	for key := range query {
		query[key] = []string{redactedValue}
	}
	redacted.RawQuery = query.Encode()
	return redacted.String()
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

var credentialJSONKeys = [...]string{
	"token",
	"access_token",
	"refresh_token",
	"id_token",
	"identity_token",
}

// containsCredentials returns true if the body contains potential credentials.
func containsCredentials(body string) bool {
	var value any
	decoder := json.NewDecoder(strings.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err == nil {
		return containsCredentialValue(value)
	}
	lowerBody := strings.ToLower(body)
	for _, key := range credentialJSONKeys {
		if strings.Contains(lowerBody, `"`+key+`"`) {
			return true
		}
	}
	return false
}

func containsCredentialValue(value any) bool {
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			if isCredentialJSONKey(key) || containsCredentialValue(child) {
				return true
			}
		}
	case []any:
		for _, child := range value {
			if containsCredentialValue(child) {
				return true
			}
		}
	}
	return false
}

func isCredentialJSONKey(key string) bool {
	for _, candidate := range credentialJSONKeys {
		if strings.EqualFold(key, candidate) {
			return true
		}
	}
	return false
}
