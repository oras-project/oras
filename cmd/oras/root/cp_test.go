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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/status"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/testutils"
)

var (
	memStore        *memory.Store
	memDesc         ocispec.Descriptor
	manifestContent = []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","artifactType":"application/vnd.unknown.artifact.v1","config":{"mediaType":"application/vnd.oci.empty.v1+json","digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2,"data":"e30="},"layers":[{"mediaType":"application/vnd.oci.empty.v1+json","digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2,"data":"e30="}]}`)
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
			w.Header().Set("Content-Length", fmt.Sprint(len(manifestContent)))
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == fmt.Sprintf("/v2/%s/manifests/%s", repoFrom, manifestDigest) &&
			r.Method == http.MethodGet:
			w.Header().Set("Content-Type", ocispec.MediaTypeImageManifest)
			w.Header().Set("Content-Length", fmt.Sprint(len(manifestContent)))
			_, _ = w.Write(manifestContent)
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
			w.Header().Set("Content-Length", fmt.Sprint(len(manifestContent)))
			_, _ = w.Write(manifestContent)
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
	reader, child, err := testutils.NewPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = child.Close() }()
	var opts copyOptions
	opts.TTY = child
	opts.From.Reference = memDesc.Digest.String()
	dst := memory.New()
	handler := status.NewTTYCopyHandler(opts.TTY)
	// test
	_, err = doCopy(context.Background(), handler, memStore, dst, &opts)
	if err != nil {
		t.Fatal(err)
	}
	// validate
	if err = testutils.MatchPipe(reader, child, "Copied", memDesc.MediaType, "100.00%", memDesc.Digest.String()); err != nil {
		t.Fatal(err)
	}
}

func Test_doCopy_skipped(t *testing.T) {
	// prepare
	reader, child, err := testutils.NewPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = child.Close() }()
	var opts copyOptions
	opts.TTY = child
	opts.From.Reference = memDesc.Digest.String()
	handler := status.NewTTYCopyHandler(opts.TTY)

	// test
	_, err = doCopy(context.Background(), handler, memStore, memStore, &opts)
	if err != nil {
		t.Fatal(err)
	}
	// validate
	if err = testutils.MatchPipe(reader, child, "Exists", memDesc.MediaType, "100.00%", memDesc.Digest.String()); err != nil {
		t.Fatal(err)
	}
}

func Test_doCopy_mounted(t *testing.T) {
	// prepare
	reader, child, err := testutils.NewPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = child.Close() }()
	var opts copyOptions
	opts.TTY = child
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
	handler := status.NewTTYCopyHandler(opts.TTY)

	// test
	_, err = doCopy(context.Background(), handler, from, to, &opts)
	if err != nil {
		t.Fatal(err)
	}
	// validate
	if err = testutils.MatchPipe(reader, child, "Mounted", configMediaType, "100.00%", configDigest); err != nil {
		t.Fatal(err)
	}
}

func Test_doCopy_mountFallback(t *testing.T) {
	// Test that copy falls back to regular upload when mount fails with 401/403
	repoFromFallback := "from-fallback"
	repoToFallback := "to-fallback"

	var uploadSessionStarted atomic.Bool
	var blobUploaded atomic.Bool

	// test server that returns 401 for mount but allows regular upload
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == fmt.Sprintf("/v2/%s/manifests/%s", repoFromFallback, manifestDigest) &&
			r.Method == http.MethodHead:
			w.Header().Set("Content-Type", ocispec.MediaTypeImageManifest)
			w.Header().Set("Content-Length", fmt.Sprint(len(manifestContent)))
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == fmt.Sprintf("/v2/%s/manifests/%s", repoFromFallback, manifestDigest) &&
			r.Method == http.MethodGet:
			w.Header().Set("Content-Type", ocispec.MediaTypeImageManifest)
			w.Header().Set("Content-Length", fmt.Sprint(len(manifestContent)))
			_, _ = w.Write(manifestContent)
		case r.URL.Path == fmt.Sprintf("/v2/%s/blobs/%s", repoFromFallback, configDigest) &&
			r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Length", fmt.Sprint(len(configContent)))
			_, _ = w.Write(configContent)
		case r.URL.Path == fmt.Sprintf("/v2/%s/manifests/%s", repoToFallback, manifestDigest) &&
			r.Method == http.MethodHead:
			w.WriteHeader(http.StatusNotFound)
		case r.URL.Path == fmt.Sprintf("/v2/%s/blobs/%s", repoToFallback, configDigest) &&
			r.Method == http.MethodHead:
			// check if blob exists before upload - not found initially
			if blobUploaded.Load() {
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set("Content-Length", fmt.Sprint(len(configContent)))
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		case r.URL.Path == fmt.Sprintf("/v2/%s/blobs/uploads/", repoToFallback) &&
			r.URL.Query().Get("mount") != "" &&
			r.Method == http.MethodPost:
			// Return 401 Unauthorized for mount requests to simulate permission denial
			w.WriteHeader(http.StatusUnauthorized)
		case r.URL.Path == fmt.Sprintf("/v2/%s/blobs/uploads/", repoToFallback) &&
			r.URL.Query().Get("mount") == "" &&
			r.Method == http.MethodPost:
			// Regular blob upload initiation - allow this
			uploadSessionStarted.Store(true)
			w.Header().Set("Location", fmt.Sprintf("/v2/%s/blobs/uploads/session123", repoToFallback))
			w.WriteHeader(http.StatusAccepted)
		case strings.HasPrefix(r.URL.Path, fmt.Sprintf("/v2/%s/blobs/uploads/", repoToFallback)) &&
			r.Method == http.MethodPut:
			// Blob upload completion
			blobUploaded.Store(true)
			w.Header().Set("Docker-Content-Digest", configDigest)
			w.WriteHeader(http.StatusCreated)
		case r.URL.Path == fmt.Sprintf("/v2/%s/manifests/%s", repoToFallback, manifestDigest) &&
			r.Method == http.MethodPut:
			w.WriteHeader(http.StatusCreated)
		case r.URL.Path == fmt.Sprintf("/v2/%s/manifests/%s", repoToFallback, manifestDigest) &&
			r.Method == http.MethodGet:
			w.Header().Set("Content-Type", ocispec.MediaTypeImageManifest)
			w.Header().Set("Content-Length", fmt.Sprint(len(manifestContent)))
			_, _ = w.Write(manifestContent)
		default:
			w.WriteHeader(http.StatusNotAcceptable)
		}
	}))
	defer ts.Close()

	uri, _ := url.Parse(ts.URL)
	testHost := "localhost:" + uri.Port()

	// prepare
	reader, child, err := testutils.NewPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = child.Close() }()
	var opts copyOptions
	opts.TTY = child
	opts.From.Reference = manifestDigest
	// mocked repositories
	from, err := remote.NewRepository(fmt.Sprintf("%s/%s", testHost, repoFromFallback))
	if err != nil {
		t.Fatal(err)
	}
	from.PlainHTTP = true
	to, err := remote.NewRepository(fmt.Sprintf("%s/%s", testHost, repoToFallback))
	if err != nil {
		t.Fatal(err)
	}
	to.PlainHTTP = true
	handler := status.NewTTYCopyHandler(opts.TTY)

	// test
	_, err = doCopy(context.Background(), handler, from, to, &opts)
	if err != nil {
		t.Fatal(err)
	}

	// validate that regular upload was used (fallback succeeded)
	if !uploadSessionStarted.Load() {
		t.Error("expected regular upload session to be started after mount failure")
	}
	if !blobUploaded.Load() {
		t.Error("expected blob to be uploaded via regular upload after mount failure")
	}
	// validate output shows "Copied" instead of "Mounted"
	if err = testutils.MatchPipe(reader, child, "Copied", configMediaType, "100.00%", configDigest); err != nil {
		t.Fatal(err)
	}
}

func Test_prepareCopyOption_nonIndex(t *testing.T) {
	ctx := context.Background()
	root := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
	}
	if _, _, err := prepareCopyOption(ctx, nil, nil, root, oras.ExtendedCopyGraphOptions{}); err != nil {
		t.Errorf("prepareCopyOption() error = %v, wantErr false", err)
	}
}

var errMockedFetch = fmt.Errorf("fetch error")

// fetchFailingReadOnlyGraphTarget is a mock implementation of oras.ReadOnlyGraphTarget
type fetchFailingReadOnlyGraphTarget struct {
	oras.ReadOnlyGraphTarget
}

// Fetch simulates a failure when fetching content from the source.
func (m *fetchFailingReadOnlyGraphTarget) Fetch(_ context.Context, _ ocispec.Descriptor) (io.ReadCloser, error) {
	return nil, errMockedFetch
}

func Test_prepareCopyOption_fetchFailure(t *testing.T) {
	ctx := context.Background()
	src := &fetchFailingReadOnlyGraphTarget{}
	dst := memory.New()
	root := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageIndex,
		Digest:    digest.FromString("nonexistent"),
		Size:      int64(len("nonexistent")),
	}

	if _, _, err := prepareCopyOption(ctx, src, dst, root, oras.ExtendedCopyGraphOptions{}); !errors.Is(err, errMockedFetch) {
		t.Errorf("prepareCopyOption() error = %v, want %v", err, errMockedFetch)
	}
}

func Test_recursiveCopy_prepareCopyOptionFailure(t *testing.T) {
	ctx := context.Background()
	src := &fetchFailingReadOnlyGraphTarget{}
	dst := memory.New()
	root := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageIndex,
		Digest:    digest.FromString("nonexistent"),
		Size:      int64(len("nonexistent")),
	}

	if _, _, err := prepareCopyOption(ctx, src, dst, root, oras.ExtendedCopyGraphOptions{}); !errors.Is(err, errMockedFetch) {
		t.Errorf("prepareCopyOption() error = %v, want %v", err, errMockedFetch)
	}
}

// invalidJSONReadOnlyGraphTarget is a mock implementation of oras.ReadOnlyGraphTarget
// that returns invalid JSON data to simulate a JSON unmarshalling failure.
type invalidJSONReadOnlyGraphTarget struct {
	oras.ReadOnlyGraphTarget
}

// Fetch simulates a successful fetch of invalid JSON data.
func (m *invalidJSONReadOnlyGraphTarget) Fetch(_ context.Context, _ ocispec.Descriptor) (io.ReadCloser, error) {
	// Return invalid JSON data
	return io.NopCloser(strings.NewReader("invalid-json")), nil
}

func Test_prepareCopyOption_jsonUnmarshalFailure(t *testing.T) {
	ctx := context.Background()
	src := &invalidJSONReadOnlyGraphTarget{}
	dst := memory.New()
	root := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageIndex,
		Digest:    digest.FromString("invalid-json"),
		Size:      int64(len("invalid-json")),
	}
	_, _, err := prepareCopyOption(ctx, src, dst, root, oras.ExtendedCopyGraphOptions{})
	var syntaxErr *json.SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Errorf("prepareCopyOption() error = %v, want json.SyntaxError", err)
	}
}

// mockReferrersFailingSource is a mock implementation of oras.ReadOnlyGraphTarget
// that simulates a failure when fetching referrers.
type mockReferrersFailingSource struct {
	oras.ReadOnlyGraphTarget
	indexContent string
}

// Fetch simulates successful fetching of index content.
func (m *mockReferrersFailingSource) Fetch(_ context.Context, _ ocispec.Descriptor) (io.ReadCloser, error) {
	// Return valid JSON data to pass the fetch step
	return io.NopCloser(strings.NewReader(m.indexContent)), nil
}

func Test_prepareCopyOption_referrersFailure(t *testing.T) {
	ctx := context.Background()
	mockedIndex := `{"schemaVersion":2,"manifests":[{"mediaType":"application/vnd.oci.image.manifest.v1+json","digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2}]}`
	src := &mockReferrersFailingSource{indexContent: mockedIndex}
	dst := memory.New()
	root := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageIndex,
		Digest:    digest.FromString(mockedIndex),
		Size:      int64(len(mockedIndex)),
	}
	errMockedReferrers := fmt.Errorf("failed to get referrers")
	opts := oras.ExtendedCopyGraphOptions{
		FindPredecessors: func(_ context.Context, _ content.ReadOnlyGraphStorage, _ ocispec.Descriptor) ([]ocispec.Descriptor, error) {
			return nil, errMockedReferrers
		},
	}

	if _, _, err := prepareCopyOption(ctx, src, dst, root, opts); !errors.Is(err, errMockedReferrers) {
		t.Errorf("prepareCopyOption() error = %v, wantErr %v", err, errMockedReferrers)
	}
}

func Test_prepareCopyOption_referrersFailureOnIndex(t *testing.T) {
	ctx := context.Background()
	mockedIndex := `{"schemaVersion":2,"manifests":[{"mediaType":"application/vnd.oci.image.manifest.v1+json","digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2}]}`
	src := &mockReferrersFailingSource{indexContent: mockedIndex}
	dst := memory.New()
	root := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageIndex,
		Digest:    digest.FromString(mockedIndex),
		Size:      int64(len(mockedIndex)),
	}
	errMockedReferrers := fmt.Errorf("failed to get referrers")
	opts := oras.ExtendedCopyGraphOptions{
		FindPredecessors: func(_ context.Context, _ content.ReadOnlyGraphStorage, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
			if desc.MediaType == ocispec.MediaTypeImageIndex {
				return nil, errMockedReferrers
			}
			entry := ocispec.Descriptor{}
			return []ocispec.Descriptor{entry}, nil
		},
	}

	if _, _, err := prepareCopyOption(ctx, src, dst, root, opts); !errors.Is(err, errMockedReferrers) {
		t.Errorf("prepareCopyOption() error = %v, wantErr %v", err, errMockedReferrers)
	}
}

func Test_prepareCopyOption_noReferrers(t *testing.T) {
	ctx := context.Background()
	mockedIndex := `{"schemaVersion":2,"manifests":[{"mediaType":"application/vnd.oci.image.manifest.v1+json","digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2}]}`
	src := &mockReferrersFailingSource{indexContent: mockedIndex}
	dst := memory.New()
	root := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageIndex,
		Digest:    digest.FromString(mockedIndex),
		Size:      int64(len(mockedIndex)),
	}
	opts := oras.ExtendedCopyGraphOptions{
		FindPredecessors: func(_ context.Context, _ content.ReadOnlyGraphStorage, _ ocispec.Descriptor) ([]ocispec.Descriptor, error) {
			return nil, nil
		},
	}

	if _, _, err := prepareCopyOption(ctx, src, dst, root, opts); err != nil {
		t.Errorf("prepareCopyOption() error = %v, wantErr false", err)
	}
}

func Test_getMountPoint(t *testing.T) {
	registry1Repo1 := &remote.Repository{}
	registry1Repo1.Reference.Registry = "localhost:5000"
	registry1Repo1.Reference.Repository = "repo1"

	registry1Repo2 := &remote.Repository{}
	registry1Repo2.Reference.Registry = "localhost:5000"
	registry1Repo2.Reference.Repository = "repo2"

	registry2Repo1 := &remote.Repository{}
	registry2Repo1.Reference.Registry = "localhost:6000"
	registry2Repo1.Reference.Repository = "repo1"

	tests := []struct {
		name         string
		src          oras.ReadOnlyGraphTarget
		dst          oras.GraphTarget
		fromUsername string
		fromPassword string
		toUsername   string
		toPassword   string
		wantRepo     bool
		wantMount    string
	}{
		{
			name:         "should mount: both remote, same registry, same credentials",
			src:          registry1Repo1,
			dst:          registry1Repo2,
			fromUsername: "user1",
			fromPassword: "pass1",
			toUsername:   "user1",
			toPassword:   "pass1",
			wantRepo:     true,
			wantMount:    "repo1",
		},
		{
			name:         "should not mount: both remote, same registry, different credentials",
			src:          registry1Repo1,
			dst:          registry1Repo2,
			fromUsername: "user1",
			fromPassword: "pass1",
			toUsername:   "user2",
			toPassword:   "pass2",
			wantRepo:     false,
			wantMount:    "",
		},
		{
			name:         "should not mount: both remote, different registries",
			src:          registry1Repo1,
			dst:          registry2Repo1,
			fromUsername: "user1",
			fromPassword: "pass1",
			toUsername:   "user3",
			toPassword:   "pass3",
			wantRepo:     false,
			wantMount:    "",
		},
		{
			name:       "should not mount: source is not remote",
			src:        memStore,
			dst:        registry1Repo1,
			toUsername: "user1",
			toPassword: "pass1",
			wantRepo:   false,
			wantMount:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &copyOptions{}
			opts.From.Username = tt.fromUsername
			opts.From.Secret = tt.fromPassword
			opts.To.Username = tt.toUsername
			opts.To.Secret = tt.toPassword

			gotMount, gotRepo := getMountPoint(tt.src, tt.dst, opts)
			if gotRepo != tt.wantRepo {
				t.Errorf("checkMount() gotRepo = %v, want %v", gotRepo, tt.wantRepo)
			}
			if gotMount != tt.wantMount {
				t.Errorf("checkMount() gotRepo = %v, want %v", gotMount, tt.wantMount)
			}
		})
	}
}

// tagFailingTarget is a mock implementation of oras.Target whose Tag always
// fails, simulating a destination registry that rejects the root tag.
type tagFailingTarget struct {
	oras.Target
}

// Tag simulates a not found failure at the destination.
func (t *tagFailingTarget) Tag(_ context.Context, _ ocispec.Descriptor, _ string) error {
	return errdef.ErrNotFound
}

func Test_recursiveCopy_tagFailure(t *testing.T) {
	ctx := context.Background()
	dst := &tagFailingTarget{Target: memory.New()}

	err := recursiveCopy(ctx, memStore, dst, "v1", memDesc, oras.DefaultExtendedCopyGraphOptions)
	if err == nil {
		t.Fatal("recursiveCopy() error = nil, wantErr true")
	}
	var copyErr *oras.CopyError
	if !errors.As(err, &copyErr) {
		t.Fatalf("recursiveCopy() error = %v, want *oras.CopyError", err)
	}
	if copyErr.Op != "Tag" {
		t.Errorf("recursiveCopy() error Op = %q, want %q", copyErr.Op, "Tag")
	}
	if copyErr.Origin != oras.CopyErrorOriginDestination {
		t.Errorf("recursiveCopy() error Origin = %v, want %v", copyErr.Origin, oras.CopyErrorOriginDestination)
	}
	if !errors.Is(err, errdef.ErrNotFound) {
		t.Errorf("recursiveCopy() error = %v, want to wrap %v", err, errdef.ErrNotFound)
	}
}

// newStrandedRootSource builds a source holding an index with a single child
// manifest, together with the FindPredecessors that a registry without
// Referrers API support produces for it: resolving the referrers tag
// `sha256-<hex>` as the digest `sha256:<hex>` answers with the index itself,
// so the index's own child comes back as its referrer.
func newStrandedRootSource(t *testing.T) (*memory.Store, ocispec.Descriptor, oras.ExtendedCopyGraphOptions) {
	t.Helper()
	ctx := context.Background()
	src := memory.New()

	configDesc := ocispec.Descriptor{
		MediaType: configMediaType,
		Digest:    digest.FromBytes(configContent),
		Size:      int64(len(configContent)),
	}
	if err := src.Push(ctx, configDesc, bytes.NewReader(configContent)); err != nil {
		t.Fatal(err)
	}

	child := []byte(fmt.Sprintf(`{"schemaVersion":2,"mediaType":%q,"config":{"mediaType":%q,"digest":%q,"size":%d},"layers":[]}`,
		ocispec.MediaTypeImageManifest, configDesc.MediaType, configDesc.Digest, configDesc.Size))
	childDesc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Digest:    digest.FromBytes(child),
		Size:      int64(len(child)),
	}
	if err := src.Push(ctx, childDesc, bytes.NewReader(child)); err != nil {
		t.Fatal(err)
	}

	index := []byte(fmt.Sprintf(`{"schemaVersion":2,"mediaType":%q,"manifests":[{"mediaType":%q,"digest":%q,"size":%d}]}`,
		ocispec.MediaTypeImageIndex, childDesc.MediaType, childDesc.Digest, childDesc.Size))
	indexDesc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageIndex,
		Digest:    digest.FromBytes(index),
		Size:      int64(len(index)),
	}
	if err := src.Push(ctx, indexDesc, bytes.NewReader(index)); err != nil {
		t.Fatal(err)
	}

	opts := oras.DefaultExtendedCopyGraphOptions
	opts.FindPredecessors = func(_ context.Context, _ content.ReadOnlyGraphStorage, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		if content.Equal(desc, indexDesc) {
			return []ocispec.Descriptor{childDesc}, nil
		}
		return nil, nil
	}
	return src, indexDesc, opts
}

// Test_recursiveCopy_strandedRoot covers the copy failure reported in
// https://github.com/oras-project/oras/issues/2148. Extended copy walks up to
// the bogus referrers, treats the child as the graph root and never copies the
// index, which used to leave the final tagging with nothing to tag.
func Test_recursiveCopy_strandedRoot(t *testing.T) {
	ctx := context.Background()
	src, indexDesc, opts := newStrandedRootSource(t)
	dst := memory.New()

	if err := recursiveCopy(ctx, src, dst, "v1", indexDesc, opts); err != nil {
		t.Fatalf("recursiveCopy() error = %v, wantErr false", err)
	}

	exists, err := dst.Exists(ctx, indexDesc)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("recursiveCopy() left the root index out of the destination")
	}
	got, err := dst.Resolve(ctx, "v1")
	if err != nil {
		t.Fatalf("Resolve() error = %v, wantErr false", err)
	}
	if !content.Equal(got, indexDesc) {
		t.Errorf("Resolve() = %v, want %v", got, indexDesc)
	}
}

var errMockedPush = errors.New("push error")

// pushFailingTarget is a mock implementation of oras.Target that rejects one
// descriptor, simulating a destination that cannot accept the root.
type pushFailingTarget struct {
	oras.Target
	failOn ocispec.Descriptor
}

// Push simulates a failure for the tracked descriptor.
func (t *pushFailingTarget) Push(ctx context.Context, desc ocispec.Descriptor, r io.Reader) error {
	if content.Equal(desc, t.failOn) {
		return errMockedPush
	}
	return t.Target.Push(ctx, desc, r)
}

func Test_recursiveCopy_rootPushFailure(t *testing.T) {
	ctx := context.Background()
	src, indexDesc, opts := newStrandedRootSource(t)
	dst := &pushFailingTarget{Target: memory.New(), failOn: indexDesc}

	err := recursiveCopy(ctx, src, dst, "v1", indexDesc, opts)
	if !errors.Is(err, errMockedPush) {
		t.Fatalf("recursiveCopy() error = %v, want to wrap %v", err, errMockedPush)
	}
	var copyErr *oras.CopyError
	if !errors.As(err, &copyErr) {
		t.Fatalf("recursiveCopy() error = %v, want *oras.CopyError", err)
	}
	if copyErr.Op != "Push" {
		t.Errorf("recursiveCopy() error Op = %q, want %q", copyErr.Op, "Push")
	}
	if copyErr.Origin != oras.CopyErrorOriginDestination {
		t.Errorf("recursiveCopy() error Origin = %v, want %v", copyErr.Origin, oras.CopyErrorOriginDestination)
	}
}

// Test_recursiveCopy_genuineRootReferrerSurvives guards the filter above
// against over-filtering: a real referrer of the root must still be copied
// even when the same FindPredecessors call also reports the index's own
// children as referrers.
func Test_recursiveCopy_genuineRootReferrerSurvives(t *testing.T) {
	ctx := context.Background()
	src := memory.New()
	push := func(blob []byte, mediaType string) ocispec.Descriptor {
		desc := ocispec.Descriptor{
			MediaType: mediaType,
			Digest:    digest.FromBytes(blob),
			Size:      int64(len(blob)),
		}
		if err := src.Push(ctx, desc, bytes.NewReader(blob)); err != nil {
			t.Fatal(err)
		}
		return desc
	}

	configDesc := push(configContent, configMediaType)
	childDesc := push([]byte(fmt.Sprintf(`{"schemaVersion":2,"mediaType":%q,"config":{"mediaType":%q,"digest":%q,"size":%d},"layers":[]}`,
		ocispec.MediaTypeImageManifest, configDesc.MediaType, configDesc.Digest, configDesc.Size)), ocispec.MediaTypeImageManifest)
	indexDesc := push([]byte(fmt.Sprintf(`{"schemaVersion":2,"mediaType":%q,"manifests":[{"mediaType":%q,"digest":%q,"size":%d}]}`,
		ocispec.MediaTypeImageIndex, childDesc.MediaType, childDesc.Digest, childDesc.Size)), ocispec.MediaTypeImageIndex)
	// A genuine referrer of the index, i.e. a manifest declaring it as subject.
	referrerDesc := push([]byte(fmt.Sprintf(`{"schemaVersion":2,"mediaType":%q,"artifactType":"application/vnd.test.referrer","config":{"mediaType":%q,"digest":%q,"size":%d},"layers":[],"subject":{"mediaType":%q,"digest":%q,"size":%d}}`,
		ocispec.MediaTypeImageManifest, configDesc.MediaType, configDesc.Digest, configDesc.Size,
		indexDesc.MediaType, indexDesc.Digest, indexDesc.Size)), ocispec.MediaTypeImageManifest)

	opts := oras.DefaultExtendedCopyGraphOptions
	opts.FindPredecessors = func(_ context.Context, _ content.ReadOnlyGraphStorage, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		if content.Equal(desc, indexDesc) {
			// The phantom child and the genuine referrer, reported together.
			return []ocispec.Descriptor{childDesc, referrerDesc}, nil
		}
		return nil, nil
	}

	dst := memory.New()
	if err := recursiveCopy(ctx, src, dst, "v1", indexDesc, opts); err != nil {
		t.Fatalf("recursiveCopy() error = %v, wantErr false", err)
	}
	for name, desc := range map[string]ocispec.Descriptor{
		"index":    indexDesc,
		"child":    childDesc,
		"referrer": referrerDesc,
	} {
		exists, err := dst.Exists(ctx, desc)
		if err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Errorf("recursiveCopy() left the %s out of the destination", name)
		}
	}
}

// Test_recursiveCopy_strandedRootWithChildReferrer covers the #1728 shape of
// https://github.com/oras-project/oras/issues/2148: the index has no referrer
// of its own, but a child does, so FindPredecessors reports the phantom
// referrers on the path where the child referrer list is non-empty.
func Test_recursiveCopy_strandedRootWithChildReferrer(t *testing.T) {
	ctx := context.Background()
	src := memory.New()
	push := func(blob []byte, mediaType string) ocispec.Descriptor {
		desc := ocispec.Descriptor{
			MediaType: mediaType,
			Digest:    digest.FromBytes(blob),
			Size:      int64(len(blob)),
		}
		if err := src.Push(ctx, desc, bytes.NewReader(blob)); err != nil {
			t.Fatal(err)
		}
		return desc
	}

	configDesc := push(configContent, configMediaType)
	childDesc := push([]byte(fmt.Sprintf(`{"schemaVersion":2,"mediaType":%q,"config":{"mediaType":%q,"digest":%q,"size":%d},"layers":[]}`,
		ocispec.MediaTypeImageManifest, configDesc.MediaType, configDesc.Digest, configDesc.Size)), ocispec.MediaTypeImageManifest)
	indexDesc := push([]byte(fmt.Sprintf(`{"schemaVersion":2,"mediaType":%q,"manifests":[{"mediaType":%q,"digest":%q,"size":%d}]}`,
		ocispec.MediaTypeImageIndex, childDesc.MediaType, childDesc.Digest, childDesc.Size)), ocispec.MediaTypeImageIndex)
	// A genuine referrer of the child, not of the index.
	referrerDesc := push([]byte(fmt.Sprintf(`{"schemaVersion":2,"mediaType":%q,"artifactType":"application/vnd.test.referrer","config":{"mediaType":%q,"digest":%q,"size":%d},"layers":[],"subject":{"mediaType":%q,"digest":%q,"size":%d}}`,
		ocispec.MediaTypeImageManifest, configDesc.MediaType, configDesc.Digest, configDesc.Size,
		childDesc.MediaType, childDesc.Digest, childDesc.Size)), ocispec.MediaTypeImageManifest)

	opts := oras.DefaultExtendedCopyGraphOptions
	opts.FindPredecessors = func(_ context.Context, _ content.ReadOnlyGraphStorage, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		switch {
		case content.Equal(desc, indexDesc):
			// Phantom: the index's own child, reported as its referrer.
			return []ocispec.Descriptor{childDesc}, nil
		case content.Equal(desc, childDesc):
			return []ocispec.Descriptor{referrerDesc}, nil
		}
		return nil, nil
	}

	dst := memory.New()
	if err := recursiveCopy(ctx, src, dst, "v1", indexDesc, opts); err != nil {
		t.Fatalf("recursiveCopy() error = %v, wantErr false", err)
	}
	for name, desc := range map[string]ocispec.Descriptor{
		"index":    indexDesc,
		"child":    childDesc,
		"referrer": referrerDesc,
	} {
		exists, err := dst.Exists(ctx, desc)
		if err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Errorf("recursiveCopy() left the %s out of the destination", name)
		}
	}
}

func newMultiPlatformSource(t *testing.T) (*memory.Store, ocispec.Descriptor, ocispec.Index) {
	t.Helper()
	ctx := context.Background()
	src := memory.New()
	config := []byte(`{}`)
	configDesc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageConfig,
		Digest:    digest.FromBytes(config),
		Size:      int64(len(config)),
	}
	if err := src.Push(ctx, configDesc, bytes.NewReader(config)); err != nil {
		t.Fatal(err)
	}

	platforms := []*ocispec.Platform{
		{OS: "linux", Architecture: "amd64"},
		{OS: "linux", Architecture: "arm", Variant: "v6"},
		{OS: "linux", Architecture: "arm", Variant: "v7"},
	}
	manifests := make([]ocispec.Descriptor, 0, len(platforms))
	for _, platform := range platforms {
		manifest := ocispec.Manifest{
			Versioned: specs.Versioned{SchemaVersion: 2},
			MediaType: ocispec.MediaTypeImageManifest,
			Annotations: map[string]string{
				"test.platform": formatPlatform(platform),
			},
			Config: configDesc,
			Layers: []ocispec.Descriptor{},
		}
		contentBytes, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		desc := ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageManifest,
			Digest:    digest.FromBytes(contentBytes),
			Size:      int64(len(contentBytes)),
			Platform:  platform,
		}
		if err := src.Push(ctx, desc, bytes.NewReader(contentBytes)); err != nil {
			t.Fatal(err)
		}
		manifests = append(manifests, desc)
	}
	index := ocispec.Index{
		Versioned: specs.Versioned{SchemaVersion: 2},
		MediaType: ocispec.MediaTypeImageIndex,
		Manifests: manifests,
	}
	indexBytes, err := json.Marshal(index)
	if err != nil {
		t.Fatal(err)
	}
	root := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageIndex,
		Digest:    digest.FromBytes(indexBytes),
		Size:      int64(len(indexBytes)),
	}
	if err := src.Push(ctx, root, bytes.NewReader(indexBytes)); err != nil {
		t.Fatal(err)
	}
	if err := src.Tag(ctx, root, "source"); err != nil {
		t.Fatal(err)
	}
	return src, root, index
}

type discardCopyHandler struct {
	status.DiscardHandler
}

func (discardCopyHandler) OnMounted(context.Context, ocispec.Descriptor) error {
	return nil
}

type discardMetadataHandler struct {
	metadata.Discard
}

func (discardMetadataHandler) OnCopied(*option.BinaryTarget, ocispec.Descriptor) error {
	return nil
}

func Test_filterManifestsByPlatform(t *testing.T) {
	src, root, _ := newMultiPlatformSource(t)
	platforms := option.Platforms{Platforms: []*ocispec.Platform{{OS: "linux", Architecture: "arm"}}}
	opts := &copyOptions{Platform: platforms}
	_, selected, err := filterManifestByPlatform(context.Background(), src, root, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 2 {
		t.Fatalf("selected %d manifests, want 2", len(selected))
	}

	opts.Platform.Platforms = []*ocispec.Platform{{OS: "linux", Architecture: "arm", Variant: "v7"}}
	_, selected, err = filterManifestByPlatform(context.Background(), src, root, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 1 || selected[0].Platform.Variant != "v7" {
		t.Fatalf("selected %#v, want only arm/v7", selected)
	}

	opts.Platform.Platforms = []*ocispec.Platform{{OS: "windows", Architecture: "amd64"}}
	if _, _, err = filterManifestByPlatform(context.Background(), src, root, opts); err == nil || !strings.Contains(err.Error(), "linux/amd64") {
		t.Fatalf("filterManifestsByPlatform() error = %v, want available platforms", err)
	}
}

func Test_doMultipleCopy_copiesFilteredIndex(t *testing.T) {
	ctx := context.Background()
	src, root, index := newMultiPlatformSource(t)
	dst := memory.New()
	opts := &copyOptions{
		Platform: option.Platforms{Platforms: []*ocispec.Platform{
			{OS: "linux", Architecture: "amd64"},
			{OS: "linux", Architecture: "arm", Variant: "v7"},
		}},
	}
	opts.From.Reference = "source"
	opts.To.Reference = "destination"
	_, selected, err := filterManifestByPlatform(ctx, src, root, opts)
	if err != nil {
		t.Fatal(err)
	}
	statusHandler := discardCopyHandler{DiscardHandler: status.NewDiscardHandler()}
	metadataHandler := discardMetadataHandler{Discard: metadata.NewDiscardHandler()}
	if err := doMultipleCopy(ctx, statusHandler, metadataHandler, src, dst, opts, root, index, selected); err != nil {
		t.Fatal(err)
	}

	gotRoot, err := dst.Resolve(ctx, "destination")
	if err != nil {
		t.Fatal(err)
	}
	if content.Equal(gotRoot, root) {
		t.Fatal("filtered copy retained the original root digest")
	}
	filteredBytes, err := content.FetchAll(ctx, dst, gotRoot)
	if err != nil {
		t.Fatal(err)
	}
	var filtered ocispec.Index
	if err := json.Unmarshal(filteredBytes, &filtered); err != nil {
		t.Fatal(err)
	}
	if len(filtered.Manifests) != 2 {
		t.Fatalf("filtered index contains %d manifests, want 2", len(filtered.Manifests))
	}
	if filtered.Manifests[1].Platform.Variant != "v7" {
		t.Fatalf("filtered index contains %#v, want arm/v7", filtered.Manifests[1].Platform)
	}
	if exists, err := dst.Exists(ctx, index.Manifests[1]); err != nil {
		t.Fatal(err)
	} else if exists {
		t.Fatal("filtered copy included the unselected arm/v6 manifest")
	}
}

func Test_copyMultiplePlatforms_allSelectionPreservesRoot(t *testing.T) {
	ctx := context.Background()
	src, root, _ := newMultiPlatformSource(t)
	dst := memory.New()
	opts := &copyOptions{
		Platform: option.Platforms{Platforms: []*ocispec.Platform{
			{OS: "linux", Architecture: "amd64"},
			{OS: "linux", Architecture: "arm"},
		}},
	}
	opts.From.Reference = "source"
	opts.To.Reference = "destination"
	statusHandler := discardCopyHandler{DiscardHandler: status.NewDiscardHandler()}
	metadataHandler := discardMetadataHandler{Discard: metadata.NewDiscardHandler()}
	if err := copyMultiplePlatforms(ctx, statusHandler, metadataHandler, src, dst, opts); err != nil {
		t.Fatal(err)
	}
	got, err := dst.Resolve(ctx, "destination")
	if err != nil {
		t.Fatal(err)
	}
	if !content.Equal(got, root) {
		t.Fatalf("destination root = %v, want original root %v", got.Digest, root.Digest)
	}
}

func Test_doMultipleCopy_recursiveCopiesSelectedReferrer(t *testing.T) {
	ctx := context.Background()
	src, root, index := newMultiPlatformSource(t)
	referrer, err := oras.PackManifest(ctx, src, oras.PackManifestVersion1_1, "application/vnd.test.signature", oras.PackManifestOptions{Subject: &index.Manifests[0]})
	if err != nil {
		t.Fatal(err)
	}
	dst := memory.New()
	opts := &copyOptions{
		Platform:  option.Platforms{Platforms: []*ocispec.Platform{{OS: "linux", Architecture: "amd64"}}},
		recursive: true,
	}
	opts.From.Reference = "source"
	opts.To.Reference = "destination"
	_, selected, err := filterManifestByPlatform(ctx, src, root, opts)
	if err != nil {
		t.Fatal(err)
	}
	statusHandler := discardCopyHandler{DiscardHandler: status.NewDiscardHandler()}
	metadataHandler := discardMetadataHandler{Discard: metadata.NewDiscardHandler()}
	if err := doMultipleCopy(ctx, statusHandler, metadataHandler, src, dst, opts, root, index, selected); err != nil {
		t.Fatal(err)
	}
	if exists, err := dst.Exists(ctx, referrer); err != nil {
		t.Fatal(err)
	} else if !exists {
		t.Fatal("recursive filtered copy did not include the selected manifest referrer")
	}
	if exists, err := dst.Exists(ctx, index.Manifests[1]); err != nil {
		t.Fatal(err)
	} else if exists {
		t.Fatal("recursive filtered copy included an unselected manifest")
	}
}
