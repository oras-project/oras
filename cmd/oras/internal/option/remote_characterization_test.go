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

package option

// The tests in this file pin how the registry clients built by the CLI behave
// today, before their construction moves to remote.ClientBuilder (#2379). They
// only use NewRepository and NewRegistry and observe what the test servers
// receive, so they do not depend on how the client is assembled.

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	remoteconfig "github.com/oras-project/oras-go/v3/registry/remote/config"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

// registryClientModes lists the ways the CLI builds a registry client. Every
// test that runs per mode must pass in each of them. A new way of building the
// client, such as the ClientBuilder path behind a feature flag (#2386), adds
// its mode here.
var registryClientModes = []struct {
	name  string
	setup func(t *testing.T)
}{
	{
		name: "default",
		setup: func(t *testing.T) {
			t.Helper()
			t.Setenv(ContainersConfigEnv, "")
		},
	},
	{
		name: "containers-config",
		setup: func(t *testing.T) {
			t.Helper()
			useContainersConfig(t, "")
		},
	},
}

const characterizationHeader = "X-Oras-Characterization"

var characterizationBlob = []byte("characterization blob")

func characterizationBlobDescriptor() ocispec.Descriptor {
	return ocispec.Descriptor{
		MediaType: "application/octet-stream",
		Digest:    digest.FromBytes(characterizationBlob),
		Size:      int64(len(characterizationBlob)),
	}
}

// seenRequest is what a test server saw of one request.
type seenRequest struct {
	host              string
	header            http.Header
	clientCertificate []byte
}

// requestLog records the requests a test server receives.
type requestLog struct {
	mu       sync.Mutex
	requests []seenRequest
}

func (l *requestLog) record(r *http.Request) {
	seen := seenRequest{host: r.Host, header: r.Header.Clone()}
	if r.TLS != nil && len(r.TLS.PeerCertificates) != 0 {
		seen.clientCertificate = r.TLS.PeerCertificates[0].Raw
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.requests = append(l.requests, seen)
}

func (l *requestLog) all() []seenRequest {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]seenRequest(nil), l.requests...)
}

// newBlobServer serves characterizationBlob for a blob request of "repo",
// or redirects it to redirect when that is not empty.
func newBlobServer(t *testing.T, useTLS bool, redirect string, log *requestLog) *httptest.Server {
	t.Helper()
	blobPath := "/v2/repo/blobs/" + digest.FromBytes(characterizationBlob).String()
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.record(r)
		switch {
		case r.URL.Path == "/redirected":
			_, _ = w.Write(characterizationBlob)
		case r.URL.Path == blobPath && redirect != "":
			http.Redirect(w, r, redirect+"/redirected", http.StatusTemporaryRedirect)
		case r.URL.Path == blobPath:
			_, _ = w.Write(characterizationBlob)
		default:
			http.NotFound(w, r)
		}
	}))
	if useTLS {
		server.TLS = loadTestingTLSConfig()
		server.StartTLS()
	} else {
		server.Start()
	}
	t.Cleanup(server.Close)
	return server
}

func fetchCharacterizationBlob(t *testing.T, opts *Remote, reference string) {
	t.Helper()
	repo, err := opts.NewRepository(reference, Common{}, logrus.New())
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	rc, err := repo.Fetch(context.Background(), characterizationBlobDescriptor())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("reading the blob: %v", err)
	}
	if !bytes.Equal(got, characterizationBlob) {
		t.Fatalf("blob = %q, want %q", got, characterizationBlob)
	}
}

func serverHost(t *testing.T, server *httptest.Server) string {
	t.Helper()
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	return u.Host
}

// localhostAddress returns the address of server under the localhost name,
// which the CLI reaches over plain HTTP by default.
func localhostAddress(t *testing.T, server *httptest.Server) string {
	t.Helper()
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	return "localhost:" + u.Port()
}

func writeTestFile(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// clientCertificateFlags returns a Remote that trusts the test server
// certificate and presents the test client certificate, as --ca-file,
// --cert-file and --key-file do.
func clientCertificateFlags(t *testing.T) Remote {
	t.Helper()
	dir := t.TempDir()
	return Remote{
		CACertFilePath: writeTestFile(t, dir, "ca.pem", localhostServerCert),
		CertFilePath:   writeTestFile(t, dir, "client.pem", localhostClientCert),
		KeyFilePath:    writeTestFile(t, dir, "client.key", testingKey(localhostClientKey)),
		plainHTTP:      plainHTTPNotSpecified,
	}
}

func localhostClientCertificateDER(t *testing.T) []byte {
	t.Helper()
	block, _ := pem.Decode(localhostClientCert)
	if block == nil {
		t.Fatal("cannot decode the test client certificate")
	}
	return block.Bytes
}

func TestCharacterization_clientCertificateScopedToRegistryOrigin(t *testing.T) {
	for _, mode := range registryClientModes {
		t.Run(mode.name, func(t *testing.T) {
			mode.setup(t)
			var registryLog, redirectLog requestLog
			redirect := newBlobServer(t, true, "", &redirectLog)
			registry := newBlobServer(t, true, redirect.URL, &registryLog)

			opts := clientCertificateFlags(t)
			fetchCharacterizationBlob(t, &opts, serverHost(t, registry)+"/repo")

			want := localhostClientCertificateDER(t)
			for _, r := range registryLog.all() {
				if !bytes.Equal(r.clientCertificate, want) {
					t.Errorf("registry request without the client certificate")
				}
			}
			redirected := redirectLog.all()
			if len(redirected) == 0 {
				t.Fatal("the blob request was not redirected")
			}
			for _, r := range redirected {
				if r.clientCertificate != nil {
					t.Errorf("client certificate sent to the redirect origin %s", r.host)
				}
			}
		})
	}
}

func TestCharacterization_customHeadersScopedToRegistryOrigin(t *testing.T) {
	for _, mode := range registryClientModes {
		t.Run(mode.name, func(t *testing.T) {
			mode.setup(t)
			var registryLog, redirectLog requestLog
			redirect := newBlobServer(t, false, "", &redirectLog)
			registry := newBlobServer(t, false, redirect.URL, &registryLog)

			opts := Remote{
				headerFlags: []string{characterizationHeader + ": secret"},
				plainHTTP:   plainHTTPEnabled,
			}
			if err := opts.parseCustomHeaders(); err != nil {
				t.Fatal(err)
			}
			fetchCharacterizationBlob(t, &opts, serverHost(t, registry)+"/repo")

			for _, r := range registryLog.all() {
				if got := strings.TrimSpace(r.header.Get(characterizationHeader)); got != "secret" {
					t.Errorf("registry request header %s = %q, want the --header value", characterizationHeader, got)
				}
			}
			redirected := redirectLog.all()
			if len(redirected) == 0 {
				t.Fatal("the blob request was not redirected")
			}
			for _, r := range redirected {
				if values := r.header.Values(characterizationHeader); len(values) != 0 {
					t.Errorf("--header sent to the redirect origin %s: %q", r.host, values)
				}
			}
		})
	}
}

func TestCharacterization_resolveOverridesDialTarget(t *testing.T) {
	for _, mode := range registryClientModes {
		t.Run(mode.name, func(t *testing.T) {
			mode.setup(t)
			var log requestLog
			server := newBlobServer(t, true, "", &log)
			u, err := url.Parse(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			const name = "registry.characterization.oras"
			opts := Remote{
				resolveFlag: []string{fmt.Sprintf("%s:%s:%s", name, u.Port(), u.Hostname())},
				Insecure:    true,
				plainHTTP:   plainHTTPNotSpecified,
			}
			fetchCharacterizationBlob(t, &opts, fmt.Sprintf("%s:%s/repo", name, u.Port()))

			requests := log.all()
			if len(requests) == 0 {
				t.Fatal("no request reached the --resolve target")
			}
			for _, r := range requests {
				if want := fmt.Sprintf("%s:%s", name, u.Port()); r.host != want {
					t.Errorf("Host = %q, want %q", r.host, want)
				}
			}
		})
	}
}

func TestCharacterization_localhostDefaultsToPlainHTTP(t *testing.T) {
	for _, mode := range registryClientModes {
		t.Run(mode.name, func(t *testing.T) {
			mode.setup(t)
			var log requestLog
			server := newBlobServer(t, false, "", &log)
			opts := Remote{plainHTTP: plainHTTPNotSpecified}
			fetchCharacterizationBlob(t, &opts, localhostAddress(t, server)+"/repo")
			if len(log.all()) == 0 {
				t.Fatal("no plain HTTP request reached localhost")
			}
		})
	}
}

// newBasicAuthServer serves the tag list of any repository to a request with
// Basic credentials and counts the requests that came without them.
func newBasicAuthServer(t *testing.T) (*httptest.Server, func() int) {
	t.Helper()
	var mu sync.Mutex
	unauthenticated := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, _, ok := r.BasicAuth(); !ok {
			mu.Lock()
			unauthenticated++
			mu.Unlock()
			w.Header().Set("WWW-Authenticate", `Basic realm="characterization"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tags":["tag"]}`))
	}))
	t.Cleanup(server.Close)
	return server, func() int {
		mu.Lock()
		defer mu.Unlock()
		return unauthenticated
	}
}

func listTags(t *testing.T, opts *Remote, common Common, reference string) {
	t.Helper()
	repo, err := opts.NewRepository(reference, common, logrus.New())
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	if err := repo.Tags(context.Background(), "", func([]string) error { return nil }); err != nil {
		t.Fatalf("Tags() error = %v", err)
	}
}

// oras cp builds the source and the destination from two Remote options of
// one invocation. The auth cache is keyed by repository, so the sharing shows
// when both name the same repository, as in copying a tag to another tag.
//
// This runs only without the containers configuration: newConfiguredRepository
// does not bind the invocation's auth cache today, so there the destination
// does not reuse the source's cache.
func TestCharacterization_copySharesAuthCacheForSameCredential(t *testing.T) {
	t.Setenv(ContainersConfigEnv, "")
	server, unauthenticated := newBasicAuthServer(t)
	host := localhostAddress(t, server)
	common := Common{authCaches: newAuthCacheStore()}

	from := Remote{Username: "user", Secret: "password", plainHTTP: plainHTTPNotSpecified}
	to := Remote{Username: "user", Secret: "password", plainHTTP: plainHTTPNotSpecified}
	listTags(t, &from, common, host+"/repo")
	listTags(t, &to, common, host+"/repo")
	if got := unauthenticated(); got != 1 {
		t.Errorf("unauthenticated requests = %d, want 1: the destination should reuse the source's auth cache", got)
	}

	other := Remote{Username: "other", Secret: "password", plainHTTP: plainHTTPNotSpecified}
	listTags(t, &other, common, host+"/repo")
	if got := unauthenticated(); got != 2 {
		t.Errorf("unauthenticated requests = %d, want 2: another credential must not reuse the cache", got)
	}
}

func TestCharacterization_registryWarningLoggedOncePerRegistry(t *testing.T) {
	for _, mode := range registryClientModes {
		t.Run(mode.name, func(t *testing.T) {
			mode.setup(t)
			newWarningServer := func() string {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Warning", `299 - "characterization warning"`)
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"tags":["tag"],"repositories":["repo"]}`))
				}))
				t.Cleanup(server.Close)
				return localhostAddress(t, server)
			}
			first, second := newWarningServer(), newWarningServer()

			logger, hook := test.NewNullLogger()
			opts := Remote{plainHTTP: plainHTTPNotSpecified}
			for _, reference := range []string{first + "/a", first + "/b", second + "/a"} {
				repo, err := opts.NewRepository(reference, Common{}, logger)
				if err != nil {
					t.Fatalf("NewRepository() error = %v", err)
				}
				if err := repo.Tags(context.Background(), "", func([]string) error { return nil }); err != nil {
					t.Fatalf("Tags() error = %v", err)
				}
			}
			reg, err := opts.NewRegistry(first, Common{}, logger)
			if err != nil {
				t.Fatalf("NewRegistry() error = %v", err)
			}
			if err := reg.Repositories(context.Background(), "", func([]string) error { return nil }); err != nil {
				t.Fatalf("Repositories() error = %v", err)
			}

			logged := map[any]int{}
			for _, entry := range hook.AllEntries() {
				logged[entry.Data["registry"]]++
			}
			if len(logged) != 2 || logged[first] != 1 || logged[second] != 1 {
				t.Errorf("warnings logged per registry = %v, want one each for %s and %s", logged, first, second)
			}
		})
	}
}

// newUnrelatedCertificate returns a self-signed certificate and its key that
// neither the test server nor the test client trusts.
func newUnrelatedCertificate(t *testing.T) (certPEM, keyPEM []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "unrelated"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
}

// useContainersConfigWithCertsDir enables the containers configuration with
// registries.conf and a certs.d directory for host holding files.
func useContainersConfigWithCertsDir(t *testing.T, registriesConf, host string, files map[string][]byte) {
	t.Helper()
	t.Setenv(ContainersConfigEnv, "true")
	dir := t.TempDir()
	hostDir := filepath.Join(dir, "certs.d", host)
	if err := os.MkdirAll(hostDir, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		writeTestFile(t, hostDir, name, content)
	}
	missing := filepath.Join(dir, "missing")
	configs, err := remoteconfig.LoadConfigsWithOptions(remoteconfig.LoadConfigsOptions{
		DockerConfigPath:     missing,
		ContainersAuthPath:   missing,
		RegistriesConfigPath: writeTestFile(t, dir, "registries.conf", []byte(registriesConf)),
		PolicyConfigPath:     missing,
		RegistriesDPath:      missing,
		CertsDirPaths:        []string{filepath.Join(dir, "certs.d")},
	})
	if err != nil {
		t.Fatal(err)
	}
	setContainersConfigLoader(t, func() (*remoteconfig.Configs, error) { return configs, nil })
}

func TestCharacterization_flagsTakePrecedenceOverCertsDir(t *testing.T) {
	unrelatedCert, unrelatedKey := newUnrelatedCertificate(t)

	t.Run("--ca-file is added to the certs.d CAs", func(t *testing.T) {
		var log requestLog
		server := newBlobServer(t, true, "", &log)
		host := serverHost(t, server)
		useContainersConfigWithCertsDir(t, "", host, map[string][]byte{"ca.crt": unrelatedCert})
		opts := Remote{
			CACertFilePath: writeTestFile(t, t.TempDir(), "ca.pem", localhostServerCert),
			plainHTTP:      plainHTTPNotSpecified,
		}
		fetchCharacterizationBlob(t, &opts, host+"/repo")
	})

	t.Run("certs.d CAs are kept with --ca-file", func(t *testing.T) {
		var log requestLog
		server := newBlobServer(t, true, "", &log)
		host := serverHost(t, server)
		useContainersConfigWithCertsDir(t, "", host, map[string][]byte{"ca.crt": localhostServerCert})
		opts := Remote{
			CACertFilePath: writeTestFile(t, t.TempDir(), "ca.pem", unrelatedCert),
			plainHTTP:      plainHTTPNotSpecified,
		}
		fetchCharacterizationBlob(t, &opts, host+"/repo")
	})

	t.Run("--cert-file and --key-file replace the certs.d client certificate", func(t *testing.T) {
		var log requestLog
		server := newBlobServer(t, true, "", &log)
		host := serverHost(t, server)
		useContainersConfigWithCertsDir(t, "", host, map[string][]byte{
			"ca.crt":      localhostServerCert,
			"client.cert": unrelatedCert,
			"client.key":  unrelatedKey,
		})
		dir := t.TempDir()
		opts := Remote{
			CertFilePath: writeTestFile(t, dir, "client.pem", localhostClientCert),
			KeyFilePath:  writeTestFile(t, dir, "client.key", testingKey(localhostClientKey)),
			plainHTTP:    plainHTTPNotSpecified,
		}
		fetchCharacterizationBlob(t, &opts, host+"/repo")
		want := localhostClientCertificateDER(t)
		for _, r := range log.all() {
			if !bytes.Equal(r.clientCertificate, want) {
				t.Error("the registry did not get the --cert-file certificate")
			}
		}
	})

	t.Run("--insecure overrides the certs.d CAs", func(t *testing.T) {
		var log requestLog
		server := newBlobServer(t, true, "", &log)
		host := serverHost(t, server)
		useContainersConfigWithCertsDir(t, "", host, map[string][]byte{"ca.crt": unrelatedCert})
		opts := Remote{Insecure: true, plainHTTP: plainHTTPNotSpecified}
		fetchCharacterizationBlob(t, &opts, host+"/repo")
	})

	t.Run("--plain-http overrides registries.conf", func(t *testing.T) {
		var log requestLog
		server := newBlobServer(t, false, "", &log)
		host := serverHost(t, server)
		useContainersConfigWithCertsDir(t, fmt.Sprintf("[[registry]]\nlocation = %q\n", host), host, map[string][]byte{"ca.crt": localhostServerCert})
		opts := Remote{plainHTTP: plainHTTPEnabled}
		fetchCharacterizationBlob(t, &opts, host+"/repo")
	})
}

// newManifestServer answers a manifest request for repo:tag, or 404s every
// request when serve is false, as a mirror without the image does.
func newManifestServer(t *testing.T, serve bool, log *requestLog) *httptest.Server {
	t.Helper()
	manifest := []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"mediaType":"application/vnd.oci.empty.v1+json","digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2},"layers":[]}`)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.record(r)
		if !serve || r.URL.Path != "/v2/repo/manifests/tag" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", ocispec.MediaTypeImageManifest)
		w.Header().Set("Docker-Content-Digest", digest.FromBytes(manifest).String())
		w.Header().Set("Content-Length", fmt.Sprint(len(manifest)))
		if r.Method == http.MethodGet {
			_, _ = w.Write(manifest)
		}
	}))
	server.TLS = loadTestingTLSConfig()
	server.StartTLS()
	t.Cleanup(server.Close)
	return server
}

func TestCharacterization_containersConfigFlagsApplyToPrimaryOnly(t *testing.T) {
	var primaryLog, mirrorLog requestLog
	primary := newManifestServer(t, true, &primaryLog)
	mirror := newManifestServer(t, false, &mirrorLog)
	primaryHost, mirrorHost := serverHost(t, primary), serverHost(t, mirror)
	useContainersConfig(t, fmt.Sprintf(`
[[registry]]
location = %q

[[registry.mirror]]
location = %q
insecure = true
`, primaryHost, mirrorHost))

	opts := clientCertificateFlags(t)
	opts.headerFlags = []string{characterizationHeader + ": secret"}
	if err := opts.parseCustomHeaders(); err != nil {
		t.Fatal(err)
	}
	repo, err := opts.NewRepository(primaryHost+"/repo", Common{}, logrus.New())
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	if _, err := repo.Resolve(context.Background(), "tag"); err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	mirrored := mirrorLog.all()
	if len(mirrored) == 0 {
		t.Fatal("the mirror was not tried")
	}
	for _, r := range mirrored {
		if values := r.header.Values(characterizationHeader); len(values) != 0 {
			t.Errorf("--header sent to the mirror: %q", values)
		}
		if r.clientCertificate != nil {
			t.Error("--cert-file certificate sent to the mirror")
		}
	}
	primaryRequests := primaryLog.all()
	if len(primaryRequests) == 0 {
		t.Fatal("the primary registry was not reached after the mirror missed")
	}
	want := localhostClientCertificateDER(t)
	for _, r := range primaryRequests {
		if len(r.header.Values(characterizationHeader)) == 0 {
			t.Error("--header not sent to the primary registry")
		}
		if !bytes.Equal(r.clientCertificate, want) {
			t.Error("--cert-file certificate not sent to the primary registry")
		}
	}
}
