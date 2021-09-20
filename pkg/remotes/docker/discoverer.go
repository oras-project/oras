/*
   Copyright The containerd Authors.

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

package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/reference"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/docker/docker/errdefs"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	artifactspec "github.com/oras-project/artifacts-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context/ctxhttp"
)

func WithDiscover(ref string, resolver remotes.Resolver, opts *docker.ResolverOptions) (remotes.Resolver, error) {
	opts = NewOpts(opts)

	r, err := reference.Parse(ref)
	if err != nil {
		return nil, err
	}

	return &dockerDiscoverer{
		header:     opts.Headers,
		hosts:      opts.Hosts,
		refspec:    r,
		reference:  ref,
		repository: strings.TrimPrefix(r.Locator, r.Hostname()+"/"),
		tracker:    docker.NewInMemoryTracker(),
		Resolver:   resolver}, nil
}

type dockerDiscoverer struct {
	hosts      docker.RegistryHosts
	header     http.Header
	refspec    reference.Spec
	reference  string
	repository string
	tracker    docker.StatusTracker
	remotes.Resolver
}

var localhostRegex = regexp.MustCompile(`(?:^localhost$)|(?:^localhost:\\d{0,5}$)`)

func (d *dockerDiscoverer) filterHosts(caps docker.HostCapabilities) (hosts []docker.RegistryHost, err error) {
	h, err := d.hosts(d.refspec.Hostname())
	if err != nil {
		return nil, err
	}

	for _, host := range h {
		if host.Capabilities.Has(caps) || localhostRegex.MatchString(host.Host) {
			hosts = append(hosts, host)
		}
	}

	return hosts, nil
}

func (d *dockerDiscoverer) Discover(ctx context.Context, desc ocispec.Descriptor, artifactType string) ([]artifactspec.Descriptor, error) {
	ctx = log.WithLogger(ctx, log.G(ctx).WithField("digest", desc.Digest))

	hosts, err := d.filterHosts(docker.HostCapabilityResolve)
	if err != nil {
		return nil, err
	}

	if len(hosts) == 0 {
		return nil, errdefs.NotFound(errors.New("no discover hosts"))
	}

	ctx, err = docker.ContextWithRepositoryScope(ctx, d.refspec, false)
	if err != nil {
		return nil, err
	}

	v := url.Values{}
	v.Set("artifactType", artifactType)
	query := "?" + v.Encode()

	var firstErr error
	for _, originalHost := range hosts {
		host := originalHost
		host.Path = strings.TrimSuffix(host.Path, "/v2") + "/oras/artifacts/v1"

		req := d.request(host, http.MethodGet, "manifests", desc.Digest.String(), "referrers")
		req.path += query
		if err := req.addNamespace(d.refspec.Hostname()); err != nil {
			return nil, err
		}

		refs, err := d.discover(ctx, req)
		if err != nil {
			// Store the error for referencing later
			if firstErr == nil {
				firstErr = err
			}
			continue // try another host
		}

		return refs, nil
	}

	return nil, firstErr
}

func (d *dockerDiscoverer) discover(ctx context.Context, req *request) ([]artifactspec.Descriptor, error) {
	resp, err := req.doWithRetries(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var registryErr docker.Errors
		if err := json.NewDecoder(resp.Body).Decode(&registryErr); err != nil || registryErr.Len() < 1 {
			return nil, errors.Errorf("unexpected status code %v: %v", req.String(), resp.Status)
		}
		return nil, errors.Errorf("unexpected status code %v: %s - Server message: %s", req.String(), resp.Status, registryErr.Error())
	}

	result := &struct {
		References []artifactspec.Descriptor `json:"references"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return nil, err
	}

	return result.References, nil
}

func isProxy(myhost, refhost string) bool {
	if refhost != myhost {
		if refhost != "docker.io" || myhost != "registry-1.docker.io" {
			return true
		}
	}
	return false
}

func (r *request) addNamespace(ns string) (err error) {
	if !isProxy(r.host.Host, ns) {
		return nil
	}
	var q url.Values
	// Parse query
	if i := strings.IndexByte(r.path, '?'); i > 0 {
		r.path = r.path[:i+1]
		q, err = url.ParseQuery(r.path[i+1:])
		if err != nil {
			return
		}
	} else {
		r.path = r.path + "?"
		q = url.Values{}
	}
	q.Add("ns", ns)

	r.path = r.path + q.Encode()

	return
}

func (d *dockerDiscoverer) request(host docker.RegistryHost, method string, ps ...string) *request {
	header := d.header.Clone()
	if header == nil {
		header = http.Header{}
	}

	for key, value := range host.Header {
		header[key] = append(header[key], value...)
	}
	parts := append([]string{"/", host.Path, d.repository}, ps...)
	p := path.Join(parts...)
	// Join strips trailing slash, re-add ending "/" if included
	if len(parts) > 0 && strings.HasSuffix(parts[len(parts)-1], "/") {
		p = p + "/"
	}
	return &request{
		method: method,
		path:   p,
		header: header,
		host:   host,
	}
}

type request struct {
	method string
	path   string
	header http.Header
	host   docker.RegistryHost
	body   func() (io.ReadCloser, error)
	size   int64
}

func (r *request) authorize(ctx context.Context, req *http.Request) error {
	// Check if has header for host
	if r.host.Authorizer != nil {
		if err := r.host.Authorizer.Authorize(ctx, req); err != nil {
			return err
		}
	}

	return nil
}

func (r *request) do(ctx context.Context) (*http.Response, error) {
	u := r.host.Scheme + "://" + r.host.Host + r.path
	req, err := http.NewRequest(r.method, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header = http.Header{} // headers need to be copied to avoid concurrent map access
	for k, v := range r.header {
		req.Header[k] = v
	}
	if r.body != nil {
		body, err := r.body()
		if err != nil {
			return nil, err
		}
		req.Body = body
		req.GetBody = r.body
		if r.size > 0 {
			req.ContentLength = r.size
		}
	}

	ctx = log.WithLogger(ctx, log.G(ctx).WithField("url", u))
	log.G(ctx).WithFields(requestFields(req)).Debug("do request")
	if err := r.authorize(ctx, req); err != nil {
		return nil, errors.Wrap(err, "failed to authorize")
	}

	var client = &http.Client{}
	if r.host.Client != nil {
		*client = *r.host.Client
	}
	if client.CheckRedirect == nil {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			return errors.Wrap(r.authorize(ctx, req), "failed to authorize redirect")
		}
	}

	resp, err := ctxhttp.Do(ctx, client, req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to do request")
	}
	log.G(ctx).WithFields(responseFields(resp)).Debug("fetch response received")
	return resp, nil
}

func (r *request) doWithRetries(ctx context.Context, responses []*http.Response) (*http.Response, error) {
	resp, err := r.do(ctx)
	if err != nil {
		return nil, err
	}

	responses = append(responses, resp)
	retry, err := r.retryRequest(ctx, responses)
	if err != nil {
		resp.Body.Close()
		return nil, err
	}
	if retry {
		resp.Body.Close()
		return r.doWithRetries(ctx, responses)
	}
	return resp, err
}

func (r *request) retryRequest(ctx context.Context, responses []*http.Response) (bool, error) {
	if len(responses) > 5 {
		return false, nil
	}
	last := responses[len(responses)-1]
	switch last.StatusCode {
	case http.StatusUnauthorized:
		log.G(ctx).WithField("header", last.Header.Get("WWW-Authenticate")).Debug("Unauthorized")
		if r.host.Authorizer != nil {
			if err := r.host.Authorizer.AddResponses(ctx, responses); err == nil {
				return true, nil
			} else if !errdefs.IsNotImplemented(err) {
				return false, err
			}
		}

		return false, nil
	case http.StatusMethodNotAllowed:
		// Support registries which have not properly implemented the HEAD method for
		// manifests endpoint
		if r.method == http.MethodHead && strings.Contains(r.path, "/manifests/") {
			r.method = http.MethodGet
			return true, nil
		}
	case http.StatusRequestTimeout, http.StatusTooManyRequests:
		return true, nil
	}

	// TODO: Handle 50x errors accounting for attempt history
	return false, nil
}

func (r *request) String() string {
	return r.host.Scheme + "://" + r.host.Host + r.path
}

func requestFields(req *http.Request) logrus.Fields {
	fields := map[string]interface{}{
		"request.method": req.Method,
	}
	for k, vals := range req.Header {
		k = strings.ToLower(k)
		if k == "authorization" {
			continue
		}
		for i, v := range vals {
			field := "request.header." + k
			if i > 0 {
				field = fmt.Sprintf("%s.%d", field, i)
			}
			fields[field] = v
		}
	}

	return logrus.Fields(fields)
}

func responseFields(resp *http.Response) logrus.Fields {
	fields := map[string]interface{}{
		"response.status": resp.Status,
	}
	for k, vals := range resp.Header {
		k = strings.ToLower(k)
		for i, v := range vals {
			field := "response.header." + k
			if i > 0 {
				field = fmt.Sprintf("%s.%d", field, i)
			}
			fields[field] = v
		}
	}

	return logrus.Fields(fields)
}
