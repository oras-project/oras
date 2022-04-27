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
	"context"
	"net/http"
	"net/http/httptrace"
	"strings"

	"github.com/sirupsen/logrus"
)

// Transport is an http.RoundTripper that keeps track of the in-flight
// request and add hooks to report HTTP tracing events.
type Transport struct {
	base     http.RoundTripper
	request  *http.Request
	response *http.Response
}

// RoundTrip calls base roundtrip while keeping track of the current request.
func (t *Transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	t.request = req
	resp, err = t.base.RoundTrip(req)
	t.response = resp
	return resp, err
}

// ConnectDone prints request information when the connection is done.
func (t *Transport) ConnectDone(ctx context.Context, network, addr string, err error) {
	e := ctx.Value(loggerKey{}).(*logrus.Entry)
	r := t.request
	e.Debugf("Done HTTPS connection: %s\n", r.Host)
}

// ConnStarted prints request information when the connection starts.
func (t *Transport) ConnStarted(ctx context.Context, network, addr string) {
	e := ctx.Value(loggerKey{}).(*logrus.Entry)
	r := t.request
	e.Debugf("Starting new HTTPS connection: %s\n", r.Host)
}

// GotConn prints the http request and response detail of the used connection.
func (t *Transport) GotConn(ctx context.Context, info httptrace.GotConnInfo) {
	e := ctx.Value(loggerKey{}).(*logrus.Entry)

	req := t.request
	resp := t.response

	e.Debugf(" Request URL: '%v'\n", req.URL)
	e.Debugf(" Request method:'%v'\n", req.Method)
	e.Debugf(" Request headers:\n")
	logHeader(req.Header, e)
	e.Debugf(" Request body:\n")

	if resp != nil {
		e.Debugf(" Response Status: '%v'\n", resp.Status)
		e.Debugf(" Response headers:\n")
		logHeader(resp.Header, e)
		e.Debugf(" Response content length: %v\n", resp.ContentLength)
	}
}

// logHeader prints out the provided header keys and values, with auth header
// scrubbed.
func logHeader(header http.Header, e *logrus.Entry) {
	if len(header) > 0 {
		for k, v := range header {
			if k == "Authorization" {
				v = append(v[0:0], "*****")
			}
			e.Debugf("   '%s': '%s'\n", k, strings.Join(v, ", "))
		}
	} else {
		e.Debugf("   There is no header\n")
	}
}
