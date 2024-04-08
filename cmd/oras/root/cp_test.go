//go:build freebsd || linux || netbsd || openbsd || solaris

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

package root

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras/cmd/oras/internal/display/status/console/testutils"
)

var (
	memStore        *memory.Store
	memDesc         ocispec.Descriptor
	manifestConent  = []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","artifactType":"application/vnd.unknown.artifact.v1","config":{"mediaType":"application/vnd.oci.empty.v1+json","digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2,"data":"e30="},"layers":[{"mediaType":"application/vnd.oci.empty.v1+json","digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2,"data":"e30="}]}`)
	manifestDigest  = "sha256:1bb053792feb8d8d590001c212f2defad9277e091d2aa868cde2879ff41abb1b"
	configContent   = []byte("{}")
	configDigest    = "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a"
	configMediaType = "application/vnd.oci.empty.v1+json"
	host            string
	repoFrom        = "from"
	repoTo          = "to"
)

func TestMain(m *testing.M) {
	// memory store for testing
	memStore = memory.New()
	content := []byte("test")
	r := bytes.NewReader(content)
	memDesc = ocispec.Descriptor{
		MediaType: "application/octet-stream",
		Digest:    digest.FromBytes(content),
		Size:      int64(len(content)),
	}
	if err := memStore.Push(context.Background(), memDesc, r); err != nil {
		fmt.Println("Setup failed:", err)
		os.Exit(1)
	}
	if err := memStore.Tag(context.Background(), memDesc, memDesc.Digest.String()); err != nil {
		fmt.Println("Setup failed:", err)
		os.Exit(1)
	}

	// test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == fmt.Sprintf("/v2/%s/manifests/%s", repoFrom, manifestDigest) &&
			r.Method == http.MethodHead:
			w.Header().Set("Content-Type", ocispec.MediaTypeImageManifest)
			w.Header().Set("Content-Length", fmt.Sprint(len(manifestConent)))
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == fmt.Sprintf("/v2/%s/manifests/%s", repoFrom, manifestDigest) &&
			r.Method == http.MethodGet:
			w.Header().Set("Content-Type", ocispec.MediaTypeImageManifest)
			w.Header().Set("Content-Length", fmt.Sprint(len(manifestConent)))
			_, _ = w.Write(manifestConent)
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == fmt.Sprintf("/v2/%s/blobs/%s", repoFrom, configDigest) &&
			r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Length", fmt.Sprint(len(configContent)))
			_, _ = w.Write(configContent)
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == fmt.Sprintf("/v2/%s/manifests/%s", repoTo, manifestDigest) &&
			r.Method == http.MethodHead:
			w.WriteHeader(http.StatusNotFound)
		case r.URL.Path == fmt.Sprintf("/v2/%s/blobs/%s", repoTo, configDigest) &&
			r.Method == http.MethodHead:
			w.WriteHeader(http.StatusNotFound)
		case r.URL.Path == fmt.Sprintf("/v2/%s/blobs/uploads/", repoTo) &&
			r.URL.Query().Get("mount") == configDigest &&
			r.URL.Query().Get("from") == repoFrom &&
			r.Method == http.MethodPost:
			w.Header().Set("Location", fmt.Sprintf("/v2/%s/blobs/%s", repoTo, configDigest))
			w.WriteHeader(http.StatusCreated)
		case r.URL.Path == fmt.Sprintf("/v2/%s/manifests/%s", repoTo, manifestDigest) &&
			r.Method == http.MethodPut:
			w.WriteHeader(http.StatusCreated)
		case r.URL.Path == fmt.Sprintf("/v2/%s/manifests/%s", repoTo, manifestDigest) &&
			r.Method == http.MethodGet:
			w.Header().Set("Content-Type", ocispec.MediaTypeImageManifest)
			w.Header().Set("Content-Length", fmt.Sprint(len(manifestConent)))
			_, _ = w.Write(manifestConent)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotAcceptable)
		}
	}))
	defer ts.Close()
	uri, _ := url.Parse(ts.URL)
	host = "localhost:" + uri.Port()
	m.Run()
}

func Test_doCopy(t *testing.T) {
	// prepare
	pty, slave, err := testutils.NewPty()
	if err != nil {
		t.Fatal(err)
	}
	defer slave.Close()
	var opts copyOptions
	opts.TTY = slave
	opts.Verbose = true
	opts.From.Reference = memDesc.Digest.String()
	dst := memory.New()
	// test
	_, err = doCopy(context.Background(), memStore, dst, &opts)
	if err != nil {
		t.Fatal(err)
	}
	// validate
	if err = testutils.MatchPty(pty, slave, "Copied", memDesc.MediaType, "100.00%", memDesc.Digest.String()); err != nil {
		t.Fatal(err)
	}
}

func Test_doCopy_skipped(t *testing.T) {
	// prepare
	pty, slave, err := testutils.NewPty()
	if err != nil {
		t.Fatal(err)
	}
	defer slave.Close()
	var opts copyOptions
	opts.TTY = slave
	opts.Verbose = true
	opts.From.Reference = memDesc.Digest.String()
	// test
	_, err = doCopy(context.Background(), memStore, memStore, &opts)
	if err != nil {
		t.Fatal(err)
	}
	// validate
	if err = testutils.MatchPty(pty, slave, "Exists", memDesc.MediaType, "100.00%", memDesc.Digest.String()); err != nil {
		t.Fatal(err)
	}
}

func Test_doCopy_mounted(t *testing.T) {
	// prepare
	pty, slave, err := testutils.NewPty()
	if err != nil {
		t.Fatal(err)
	}
	defer slave.Close()
	var opts copyOptions
	opts.TTY = slave
	opts.Verbose = true
	opts.From.Reference = manifestDigest
	// mocked repositories
	from, err := remote.NewRepository(fmt.Sprintf("%s/%s", host, repoFrom))
	if err != nil {
		t.Fatal(err)
	}
	from.PlainHTTP = true
	to, err := remote.NewRepository(fmt.Sprintf("%s/%s", host, repoTo))
	if err != nil {
		t.Fatal(err)
	}
	to.PlainHTTP = true
	// test
	_, err = doCopy(context.Background(), from, to, &opts)
	if err != nil {
		t.Fatal(err)
	}
	// validate
	if err = testutils.MatchPty(pty, slave, "Mounted", configMediaType, "100.00%", configDigest); err != nil {
		t.Fatal(err)
	}
}
