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
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/opencontainers/go-digest"
	specs "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/oras-project/oras-go/v3"
	"github.com/oras-project/oras-go/v3/content"
	"github.com/oras-project/oras-go/v3/content/oci"
	"github.com/oras-project/oras-go/v3/errdef"
	"github.com/oras-project/oras-go/v3/registry/remote"
	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/internal/output"
	"oras.land/oras/internal/docker"
	"oras.land/oras/internal/graph"
)

const testLayerType = "application/vnd.test.layer"

// layoutBuilder writes an OCI image layout to a directory by hand, so a test
// controls every byte, including the corrupt ones.
type layoutBuilder struct {
	t     *testing.T
	dir   string
	index ocispec.Index
}

func newLayoutBuilder(t *testing.T) *layoutBuilder {
	t.Helper()
	b := &layoutBuilder{t: t, dir: t.TempDir()}
	b.index.SchemaVersion = 2
	if err := os.MkdirAll(filepath.Join(b.dir, "blobs", "sha256"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(b.dir, ocispec.ImageLayoutFile), []byte(`{"imageLayoutVersion":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return b
}

// writeBlob replaces the content of a stored blob, and fails the test if it cannot.
func (b *layoutBuilder) writeBlob(desc ocispec.Descriptor, data []byte) {
	b.t.Helper()
	root, err := os.OpenRoot(b.dir)
	if err != nil {
		b.t.Fatal(err)
	}
	defer root.Close()
	name := filepath.Join("blobs", desc.Digest.Algorithm().String(), desc.Digest.Encoded())
	f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		b.t.Fatal(err)
	}
	if _, err := f.Write(data); err != nil {
		b.t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		b.t.Fatal(err)
	}
}

func (b *layoutBuilder) blobPath(desc ocispec.Descriptor) string {
	return filepath.Join(b.dir, "blobs", desc.Digest.Algorithm().String(), desc.Digest.Encoded())
}

// blob stores content and returns its descriptor.
func (b *layoutBuilder) blob(mediaType string, data []byte) ocispec.Descriptor {
	b.t.Helper()
	desc := ocispec.Descriptor{MediaType: mediaType, Digest: digest.FromBytes(data), Size: int64(len(data))}
	if err := os.WriteFile(b.blobPath(desc), data, 0o644); err != nil {
		b.t.Fatal(err)
	}
	return desc
}

func (b *layoutBuilder) json(mediaType string, v any) ocispec.Descriptor {
	b.t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		b.t.Fatal(err)
	}
	return b.blob(mediaType, data)
}

// image stores a manifest with one config and the given layers.
func (b *layoutBuilder) image(subject *ocispec.Descriptor, layers ...ocispec.Descriptor) (manifest, config ocispec.Descriptor) {
	b.t.Helper()
	config = b.blob("application/vnd.test.config", []byte(`{"k":"v"}`))
	manifest = b.json(ocispec.MediaTypeImageManifest, ocispec.Manifest{
		Versioned: specVersion, MediaType: ocispec.MediaTypeImageManifest,
		Config: config, Layers: layers, Subject: subject,
	})
	return manifest, config
}

func (b *layoutBuilder) writeIndex() {
	b.t.Helper()
	data, err := json.Marshal(b.index)
	if err != nil {
		b.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(b.dir, ocispec.ImageIndexFile), data, 0o644); err != nil {
		b.t.Fatal(err)
	}
}

func (b *layoutBuilder) addIndex(desc ocispec.Descriptor) {
	b.t.Helper()
	b.index.Manifests = append(b.index.Manifests, desc)
	b.writeIndex()
}

func (b *layoutBuilder) tag(desc ocispec.Descriptor, name string) {
	b.t.Helper()
	desc.Annotations = map[string]string{ocispec.AnnotationRefName: name}
	b.addIndex(desc)
}

// tarball packs the layout into a tar archive and returns its path.
func (b *layoutBuilder) tarball() string {
	b.t.Helper()
	path := filepath.Join(b.t.TempDir(), "layout.tar")
	f, err := os.Create(path)
	if err != nil {
		b.t.Fatal(err)
	}
	defer f.Close()
	tw := tar.NewWriter(f)
	defer tw.Close()
	fsys := os.DirFS(b.dir)
	err = fs.WalkDir(fsys, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return err
		}
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
			return err
		}
		_, err = tw.Write(data)
		return err
	})
	if err != nil {
		b.t.Fatal(err)
	}
	return path
}

var specVersion = specs.Versioned{SchemaVersion: 2}

// runLayoutCheck checks a reference in a layout directory or tar archive.
func runLayoutCheck(t *testing.T, layoutPath, ref string, platform *ocispec.Platform) (string, error) {
	t.Helper()
	layout, err := newCheckLayout(layoutPath)
	if err != nil {
		t.Fatal(err)
	}
	return runCheckOn(context.Background(), layout, ref, true, platform)
}

func runWholeLayoutCheck(t *testing.T, layoutPath string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	printer := output.NewPrinter(&out, &out)
	printer.Verbose = true
	err := doCheckWholeLayout(context.Background(), layoutPath, 0, printer)
	return out.String(), err
}

func runCheckOn(ctx context.Context, src oras.ReadOnlyTarget, ref string, isLayout bool, platform *ocispec.Platform) (string, error) {
	var out bytes.Buffer
	resolveOpts := oras.DefaultResolveOptions
	resolveOpts.TargetPlatform = platform
	err := doCheck(ctx, src, ref, resolveOpts, isLayout, 0, output.NewPrinter(&out, &out))
	return out.String(), err
}

// summary returns the lines of the output that are not per-node status lines,
// which are written in the order the nodes finish.
func summary(out string) []string {
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		for _, prefix := range []string{"Checked ", "Layout problems:", "Missing:", "Corrupt:", "Invalid:", "Unable to check:", "Warning:", "Not checked:", "Note:"} {
			if strings.HasPrefix(line, prefix) {
				lines = append(lines, line)
				break
			}
		}
	}
	return lines
}

func wantFailure(t *testing.T, out string, err error, class string) {
	t.Helper()
	if err == nil {
		t.Fatalf("check passed, want a failure\n%s", out)
	}
	if errors.Is(err, errdef.ErrNotFound) {
		t.Errorf("the summary error wraps ErrNotFound, so it would be shown as a registry error: %v", err)
	}
	if got := strings.Count(out, class+":"); got != 1 {
		t.Errorf("found %d %q lines, want 1\n%s", got, class, out)
	}
}

func Test_check_intactArtifact(t *testing.T) {
	b := newLayoutBuilder(t)
	l1 := b.blob(testLayerType, []byte("layer one"))
	l2 := b.blob(testLayerType, []byte("layer two"))
	manifest, _ := b.image(nil, l1, l2)
	b.tag(manifest, "v1")

	out, err := runLayoutCheck(t, b.dir, "v1", nil)
	if err != nil {
		t.Fatalf("check failed on an intact artifact: %v\n%s", err, out)
	}
	if got, want := summary(out), []string{"Checked 4 nodes: 4 verified, 0 failed"}; strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("summary = %q, want %q", got, want)
	}
}

func Test_check_contentFailures(t *testing.T) {
	tests := []struct {
		name   string
		damage func(b *layoutBuilder, layer ocispec.Descriptor)
		class  string
	}{
		{"missing blob", func(b *layoutBuilder, layer ocispec.Descriptor) { os.Remove(b.blobPath(layer)) }, "Missing"},
		{"same size, other bytes", func(b *layoutBuilder, layer ocispec.Descriptor) {
			b.writeBlob(layer, bytes.Repeat([]byte("x"), int(layer.Size)))
		}, "Corrupt"},
		{"truncated", func(b *layoutBuilder, layer ocispec.Descriptor) {
			b.writeBlob(layer, []byte("lay"))
		}, "Corrupt"},
		{"trailing data", func(b *layoutBuilder, layer ocispec.Descriptor) {
			b.writeBlob(layer, []byte("layer one and more"))
		}, "Corrupt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newLayoutBuilder(t)
			layer := b.blob(testLayerType, []byte("layer one"))
			manifest, _ := b.image(nil, layer)
			b.tag(manifest, "v1")
			tt.damage(b, layer)

			out, err := runLayoutCheck(t, b.dir, "v1", nil)
			wantFailure(t, out, err, tt.class)
			if !strings.Contains(out, "Checked 3 nodes: 2 verified, 1 failed") {
				t.Errorf("unexpected summary\n%s", out)
			}
		})
	}
}

func Test_check_manifestFailures(t *testing.T) {
	t.Run("corrupt root manifest", func(t *testing.T) {
		b := newLayoutBuilder(t)
		layer := b.blob(testLayerType, []byte("layer"))
		manifest, _ := b.image(nil, layer)
		b.tag(manifest, "v1")
		data, _ := os.ReadFile(b.blobPath(manifest))
		data[len(data)/2] ^= 0x01 // same size, different bytes
		b.writeBlob(manifest, data)

		out, err := runLayoutCheck(t, b.dir, "v1", nil)
		wantFailure(t, out, err, "Corrupt")
		if !strings.Contains(out, manifest.Digest.String()) {
			t.Errorf("the failed root manifest is not named\n%s", out)
		}
	})
	t.Run("valid digest but not JSON", func(t *testing.T) {
		b := newLayoutBuilder(t)
		manifest := b.blob(ocispec.MediaTypeImageManifest, []byte("this is not json"))
		b.tag(manifest, "v1")
		out, err := runLayoutCheck(t, b.dir, "v1", nil)
		wantFailure(t, out, err, "Invalid")
	})
	t.Run("image manifest described as an index", func(t *testing.T) {
		b := newLayoutBuilder(t)
		layer := b.blob(testLayerType, []byte("layer"))
		manifest, _ := b.image(nil, layer)
		os.Remove(b.blobPath(layer)) // would be found missing, if the layers were checked
		wrong := manifest
		wrong.MediaType = ocispec.MediaTypeImageIndex
		b.tag(wrong, "v1")
		out, err := runLayoutCheck(t, b.dir, "v1", nil)
		wantFailure(t, out, err, "Invalid")
		if strings.Contains(out, "Missing:") {
			t.Errorf("the layers were checked although the manifest is invalid\n%s", out)
		}
	})
	t.Run("body media type differs from the descriptor", func(t *testing.T) {
		b := newLayoutBuilder(t)
		manifest := b.json(ocispec.MediaTypeImageManifest, map[string]any{
			"schemaVersion": 2, "mediaType": ocispec.MediaTypeImageIndex, "manifests": []any{},
		})
		b.tag(manifest, "v1")
		out, err := runLayoutCheck(t, b.dir, "v1", nil)
		wantFailure(t, out, err, "Invalid")
	})
	t.Run("manifest without a mediaType field is accepted", func(t *testing.T) {
		b := newLayoutBuilder(t)
		config := b.blob("application/vnd.test.config", []byte(`{}`))
		manifest := b.json(ocispec.MediaTypeImageManifest, map[string]any{"schemaVersion": 2, "config": config, "layers": []any{}})
		b.tag(manifest, "v1")
		if out, err := runLayoutCheck(t, b.dir, "v1", nil); err != nil {
			t.Fatalf("check failed: %v\n%s", err, out)
		}
	})
}

func Test_check_nestedIndexAndForeignLayers(t *testing.T) {
	b := newLayoutBuilder(t)
	l1 := b.blob(testLayerType, []byte("one"))
	m1, _ := b.image(nil, l1)
	foreign := ocispec.Descriptor{MediaType: "application/vnd.oci.image.layer.nondistributable.v1.tar", Digest: digest.FromString("elsewhere"), Size: 9}
	config := b.blob("application/vnd.test.config", []byte(`{"n":2}`))
	m2 := b.json(ocispec.MediaTypeImageManifest, ocispec.Manifest{Versioned: specVersion, Config: config, Layers: []ocispec.Descriptor{foreign}})
	idx := b.json(ocispec.MediaTypeImageIndex, ocispec.Index{Versioned: specVersion, Manifests: []ocispec.Descriptor{m1, m2}})
	b.tag(idx, "v1")

	out, err := runLayoutCheck(t, b.dir, "v1", nil)
	if err != nil {
		t.Fatalf("check failed: %v\n%s", err, out)
	}
	want := []string{"Checked 6 nodes: 6 verified, 0 failed", "Not checked: 1 foreign layers, which are not stored in the target"}
	if got := summary(out); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("summary = %q, want %q", got, want)
	}
}

func Test_check_subject(t *testing.T) {
	// subjectOf builds a referrer manifest that points to subject.
	t.Run("absent subject is a warning", func(t *testing.T) {
		b := newLayoutBuilder(t)
		subject := ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest, Digest: digest.FromString("not stored"), Size: 10}
		manifest, _ := b.image(&subject)
		b.tag(manifest, "v1")
		out, err := runLayoutCheck(t, b.dir, "v1", nil)
		if err != nil {
			t.Fatalf("check failed on an absent subject: %v\n%s", err, out)
		}
		got := summary(out)
		if len(got) != 2 || got[0] != "Checked 2 nodes: 2 verified, 0 failed" || !strings.HasPrefix(got[1], "Warning: the subject "+subject.Digest.String()) {
			t.Errorf("summary = %q, want 2 verified nodes and a warning for the subject", got)
		}
	})
	t.Run("corrupt subject fails", func(t *testing.T) {
		b := newLayoutBuilder(t)
		subject, _ := b.image(nil)
		b.writeBlob(subject, bytes.Repeat([]byte("x"), int(subject.Size)))
		manifest, _ := b.image(&subject)
		b.tag(manifest, "v1")
		out, err := runLayoutCheck(t, b.dir, "v1", nil)
		wantFailure(t, out, err, "Corrupt")
	})
	t.Run("present subject with a missing layer fails", func(t *testing.T) {
		b := newLayoutBuilder(t)
		layer := b.blob(testLayerType, []byte("subject layer"))
		subject, _ := b.image(nil, layer)
		os.Remove(b.blobPath(layer))
		manifest, _ := b.image(&subject)
		b.tag(manifest, "v1")
		out, err := runLayoutCheck(t, b.dir, "v1", nil)
		wantFailure(t, out, err, "Missing")
	})
	t.Run("absent subject that an index also lists fails", func(t *testing.T) {
		b := newLayoutBuilder(t)
		subject := ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest, Digest: digest.FromString("not stored"), Size: 10}
		manifest, _ := b.image(&subject)
		idx := b.json(ocispec.MediaTypeImageIndex, ocispec.Index{Versioned: specVersion, Manifests: []ocispec.Descriptor{manifest, subject}})
		b.tag(idx, "v1")
		out, err := runLayoutCheck(t, b.dir, "v1", nil)
		wantFailure(t, out, err, "Missing")
	})
	t.Run("absent subject that was asked for fails", func(t *testing.T) {
		b := newLayoutBuilder(t)
		missing := ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest, Digest: digest.FromString("absent root"), Size: 10}
		b.tag(missing, "v1")
		out, err := runLayoutCheck(t, b.dir, "v1", nil)
		wantFailure(t, out, err, "Missing")
	})
}

func Test_check_platform(t *testing.T) {
	b := newLayoutBuilder(t)
	amd := b.blob(testLayerType, []byte("amd64 layer"))
	arm := b.blob(testLayerType, []byte("arm64 layer"))
	mAmd, cAmd := b.image(nil, amd)
	configArm := b.json("application/vnd.test.config", ocispec.Image{Platform: ocispec.Platform{OS: "linux", Architecture: "arm64"}})
	mArm := b.json(ocispec.MediaTypeImageManifest, ocispec.Manifest{Versioned: specVersion, Config: configArm, Layers: []ocispec.Descriptor{arm}})
	// the amd64 config describes amd64
	_ = cAmd
	b.writeBlob(cAmd, []byte(`{"k":"v"}`))
	idx := b.json(ocispec.MediaTypeImageIndex, ocispec.Index{Versioned: specVersion, Manifests: []ocispec.Descriptor{
		withPlatform(mAmd, "linux", "amd64"), withPlatform(mArm, "linux", "arm64")}})
	b.tag(idx, "v1")
	b.writeBlob(amd, bytes.Repeat([]byte("x"), int(amd.Size))) // corrupt the amd64 layer

	t.Run("the whole index fails", func(t *testing.T) {
		out, err := runLayoutCheck(t, b.dir, "v1", nil)
		wantFailure(t, out, err, "Corrupt")
	})
	t.Run("the arm64 image alone passes", func(t *testing.T) {
		out, err := runLayoutCheck(t, b.dir, "v1", &ocispec.Platform{OS: "linux", Architecture: "arm64"})
		if err != nil {
			t.Fatalf("check failed: %v\n%s", err, out)
		}
	})
	t.Run("no matching platform", func(t *testing.T) {
		_, err := runLayoutCheck(t, b.dir, "v1", &ocispec.Platform{OS: "linux", Architecture: "s390x"})
		if err == nil || !strings.Contains(err.Error(), "failed to resolve") {
			t.Errorf("error = %v, want a resolve error", err)
		}
	})
	t.Run("a missing config is a resolve error", func(t *testing.T) {
		// without a platform in the index, resolving reads each config
		b := newLayoutBuilder(t)
		layer := b.blob(testLayerType, []byte("layer"))
		config := b.json("application/vnd.test.config", ocispec.Image{Platform: ocispec.Platform{OS: "linux", Architecture: "arm64"}})
		m := b.json(ocispec.MediaTypeImageManifest, ocispec.Manifest{Versioned: specVersion, Config: config, Layers: []ocispec.Descriptor{layer}})
		idx := b.json(ocispec.MediaTypeImageIndex, ocispec.Index{Versioned: specVersion, Manifests: []ocispec.Descriptor{m}})
		b.tag(idx, "v1")
		os.Remove(b.blobPath(config))
		_, err := runLayoutCheck(t, b.dir, "v1", &ocispec.Platform{OS: "linux", Architecture: "arm64"})
		if err == nil || !strings.Contains(err.Error(), "failed to resolve") {
			t.Errorf("error = %v, want a resolve error", err)
		}
	})
}

func withPlatform(desc ocispec.Descriptor, os, arch string) ocispec.Descriptor {
	desc.Platform = &ocispec.Platform{OS: os, Architecture: arch}
	return desc
}

func Test_check_intactTagNextToCorruptTag(t *testing.T) {
	b := newLayoutBuilder(t)
	good, _ := b.image(nil, b.blob(testLayerType, []byte("good layer")))
	bad, _ := b.image(nil, b.blob(testLayerType, []byte("bad layer")))
	b.tag(good, "good")
	b.tag(bad, "bad")
	b.writeBlob(bad, bytes.Repeat([]byte("x"), int(bad.Size)))

	for name, path := range map[string]string{"directory": b.dir, "tar archive": b.tarball()} {
		t.Run(name, func(t *testing.T) {
			if out, err := runLayoutCheck(t, path, "good", nil); err != nil {
				t.Errorf("the intact tag failed: %v\n%s", err, out)
			}
			out, err := runLayoutCheck(t, path, "bad", nil)
			wantFailure(t, out, err, "Corrupt")
		})
	}
}

func Test_check_outputOrderIsStable(t *testing.T) {
	b := newLayoutBuilder(t)
	var layers []ocispec.Descriptor
	for i := 0; i < 12; i++ {
		layer := b.blob(testLayerType, []byte(fmt.Sprintf("layer number %d", i)))
		layers = append(layers, layer)
		switch i % 3 {
		case 0:
			os.Remove(b.blobPath(layer))
		case 1:
			b.writeBlob(layer, bytes.Repeat([]byte("x"), int(layer.Size)))
		}
	}
	manifest, _ := b.image(nil, layers...)
	b.tag(manifest, "v1")

	var first []string
	for i := 0; i < 8; i++ {
		out, _ := runLayoutCheck(t, b.dir, "v1", nil)
		got := summary(out)
		if first == nil {
			first = got
			continue
		}
		if strings.Join(got, "\n") != strings.Join(first, "\n") {
			t.Fatalf("run %d differs:\n%s\nwant:\n%s", i, strings.Join(got, "\n"), strings.Join(first, "\n"))
		}
	}
	if len(first) != 9 { // the count, 4 missing and 4 corrupt
		t.Errorf("summary has %d lines, want 9: %q", len(first), first)
	}
}

// cancellingStorage cancels the context when the first blob is fetched.
type cancellingStorage struct {
	oras.ReadOnlyTarget
	cancel context.CancelFunc
}

func (s cancellingStorage) Fetch(ctx context.Context, target ocispec.Descriptor) (io.ReadCloser, error) {
	if target.MediaType == testLayerType {
		s.cancel()
	}
	return s.ReadOnlyTarget.Fetch(ctx, target)
}

func Test_check_cancellation(t *testing.T) {
	b := newLayoutBuilder(t)
	manifest, _ := b.image(nil, b.blob(testLayerType, []byte("layer")))
	b.tag(manifest, "v1")
	layout, err := newCheckLayout(b.dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out, err := runCheckOn(ctx, cancellingStorage{ReadOnlyTarget: layout, cancel: cancel}, "v1", true, nil)
	if err == nil || !strings.Contains(err.Error(), "check stopped before it finished") || !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want a stopped check that wraps context.Canceled", err)
	}
	if got := summary(out); len(got) != 0 {
		t.Errorf("a stopped check printed a summary: %q", got)
	}
}

func Test_check_layoutErrors(t *testing.T) {
	if _, err := newCheckLayout(filepath.Join(t.TempDir(), "nothing")); err == nil {
		t.Error("a missing path was accepted")
	}
	if _, err := newCheckLayout(t.TempDir()); err == nil {
		t.Error("a directory without index.json was accepted")
	}
	b := newLayoutBuilder(t)
	b.tag(ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest, Digest: digest.FromString("other"), Size: 5}, "other")
	if out, err := runLayoutCheck(t, b.dir, "v1", nil); err == nil || !errors.Is(err, errdef.ErrNotFound) {
		t.Errorf("an unknown tag: error = %v, out = %q, want not found", err, out)
	}
}

// fakeRegistry serves one artifact from memory and lets a test change how a
// blob is served.
type fakeRegistry struct {
	manifest ocispec.Descriptor
	body     map[digest.Digest][]byte
	blob     func(w http.ResponseWriter, r *http.Request, desc digest.Digest) // overrides how blobs are served
}

func (f *fakeRegistry) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	kind, ref := parts[len(parts)-2], parts[len(parts)-1]
	switch {
	case r.URL.Path == "/v2/" || r.URL.Path == "/v2":
		w.WriteHeader(http.StatusOK)
	case kind == "manifests":
		data := f.body[f.manifest.Digest]
		if ref != "v1" && ref != f.manifest.Digest.String() {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", f.manifest.MediaType)
		w.Header().Set("Docker-Content-Digest", f.manifest.Digest.String())
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		if r.Method == http.MethodGet {
			_, _ = w.Write(data)
		}
	case kind == "blobs":
		dgst := digest.Digest(ref)
		if f.blob != nil {
			f.blob(w, r, dgst)
			return
		}
		data, ok := f.body[dgst]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		if r.Method == http.MethodGet {
			_, _ = w.Write(data)
		}
	default:
		http.NotFound(w, r)
	}
}

func Test_check_registry(t *testing.T) {
	layerData := []byte("registry layer data")
	configData := []byte(`{"k":"v"}`)
	layer := ocispec.Descriptor{MediaType: testLayerType, Digest: digest.FromBytes(layerData), Size: int64(len(layerData))}
	config := ocispec.Descriptor{MediaType: "application/vnd.test.config", Digest: digest.FromBytes(configData), Size: int64(len(configData))}
	manifestData, _ := json.Marshal(ocispec.Manifest{Versioned: specVersion, MediaType: ocispec.MediaTypeImageManifest, Config: config, Layers: []ocispec.Descriptor{layer}})
	manifest := ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest, Digest: digest.FromBytes(manifestData), Size: int64(len(manifestData))}
	bodies := map[digest.Digest][]byte{layer.Digest: layerData, config.Digest: configData, manifest.Digest: manifestData}

	serveLayer := func(f func(w http.ResponseWriter, r *http.Request)) func(http.ResponseWriter, *http.Request, digest.Digest) {
		return func(w http.ResponseWriter, r *http.Request, d digest.Digest) {
			if d == layer.Digest {
				f(w, r)
				return
			}
			_, _ = w.Write(bodies[d])
		}
	}
	tests := []struct {
		name  string
		blob  func(http.ResponseWriter, *http.Request, digest.Digest)
		class string // empty means the check passes
	}{
		{"intact", nil, ""},
		{"forbidden", serveLayer(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) }), "Unable to check"},
		{"unauthorized", serveLayer(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusUnauthorized) }), "Unable to check"},
		{"server error", serveLayer(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadGateway) }), "Unable to check"},
		{"not found", serveLayer(func(w http.ResponseWriter, _ *http.Request) { http.NotFound(w, nil) }), "Missing"},
		{"other bytes of the same size", serveLayer(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Length", strconv.Itoa(len(layerData)))
			_, _ = w.Write(bytes.Repeat([]byte("x"), len(layerData)))
		}), "Corrupt"},
		{"dropped connection", serveLayer(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Length", strconv.Itoa(len(layerData)))
			_, _ = w.Write(layerData[:4])
			conn, _, _ := w.(http.Hijacker).Hijack()
			conn.Close()
		}), "Unable to check"},
		{"short blob with Content-Length", serveLayer(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Length", "4")
			_, _ = w.Write(layerData[:4])
		}), "Unable to check"},
		{"short blob without Content-Length", serveLayer(func(w http.ResponseWriter, _ *http.Request) {
			w.(http.Flusher).Flush()
			_, _ = w.Write(layerData[:4])
		}), "Unable to check"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := &fakeRegistry{manifest: manifest, body: bodies, blob: tt.blob}
			server := httptest.NewServer(reg)
			defer server.Close()
			host := strings.TrimPrefix(server.URL, "http://")
			repo, err := remote.NewRepository(host + "/test/artifact")
			if err != nil {
				t.Fatal(err)
			}
			repo.Registry.PlainHTTP = true

			out, err := runCheckOn(context.Background(), repo, "v1", false, nil)
			if tt.class == "" {
				if err != nil {
					t.Fatalf("check failed: %v\n%s", err, out)
				}
				return
			}
			wantFailure(t, out, err, tt.class)
		})
	}
}

func Test_check_rejectsCorruptRootManifestFromRegistry(t *testing.T) {
	data := []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json"}`)
	manifest := ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest, Digest: digest.FromBytes(data), Size: int64(len(data))}
	served := bytes.Repeat([]byte("x"), len(data))
	reg := &fakeRegistry{manifest: manifest, body: map[digest.Digest][]byte{manifest.Digest: served}}
	server := httptest.NewServer(reg)
	defer server.Close()
	repo, err := remote.NewRepository(strings.TrimPrefix(server.URL, "http://") + "/test/artifact")
	if err != nil {
		t.Fatal(err)
	}
	repo.Registry.PlainHTTP = true
	out, err := runCheckOn(context.Background(), repo, "v1", false, nil)
	if err == nil {
		t.Fatalf("check passed on a corrupt root manifest\n%s", out)
	}
}

func Test_checkManifestShape(t *testing.T) {
	tests := []struct {
		name      string
		mediaType string
		body      string
		wantErr   bool
	}{
		{"image manifest", ocispec.MediaTypeImageManifest, `{"schemaVersion":2,"config":{},"layers":[]}`, false},
		{"image manifest with its media type", ocispec.MediaTypeImageManifest, `{"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{}}`, false},
		{"docker manifest", docker.MediaTypeManifest, `{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{},"layers":[]}`, false},
		{"artifact manifest", graph.MediaTypeArtifactManifest, `{"mediaType":"application/vnd.oci.artifact.manifest.v1+json","blobs":[]}`, false},
		{"artifact manifest without blobs", graph.MediaTypeArtifactManifest, `{"mediaType":"application/vnd.oci.artifact.manifest.v1+json"}`, false},
		{"artifact manifest with no shape", graph.MediaTypeArtifactManifest, `{}`, true},
		{"index", ocispec.MediaTypeImageIndex, `{"schemaVersion":2,"manifests":[]}`, false},
		{"docker manifest list", docker.MediaTypeManifestList, `{"schemaVersion":2,"manifests":[]}`, false},
		{"image manifest with an index body", ocispec.MediaTypeImageManifest, `{"manifests":[]}`, true},
		{"index with an image manifest body", ocispec.MediaTypeImageIndex, `{"config":{},"layers":[]}`, true},
		{"index with both shapes", ocispec.MediaTypeImageIndex, `{"manifests":[],"config":{}}`, true},
		{"image manifest with no shape", ocispec.MediaTypeImageManifest, `{}`, true},
		{"index with no shape", ocispec.MediaTypeImageIndex, `{"schemaVersion":2}`, true},
		{"wrong mediaType field", ocispec.MediaTypeImageManifest, `{"mediaType":"application/vnd.oci.image.index.v1+json","config":{}}`, true},
		{"not JSON", ocispec.MediaTypeImageManifest, `not json`, true},
		{"JSON array", ocispec.MediaTypeImageManifest, `[]`, true},
		{"empty body", ocispec.MediaTypeImageManifest, ``, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkManifestShape(tt.mediaType, []byte(tt.body))
			if (err != nil) != tt.wantErr {
				t.Errorf("checkManifestShape() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_checker_classify(t *testing.T) {
	var v any
	syntaxErr := fmt.Errorf("failed to read sha256:abc: %w", json.Unmarshal([]byte("{"), &v))
	typeErr := fmt.Errorf("failed to read sha256:abc: %w", json.Unmarshal([]byte(`{"layers":3}`), &ocispec.Manifest{}))
	tests := []struct {
		name     string
		err      error
		isLayout bool
		want     checkClass
	}{
		{"not found", fmt.Errorf("failed to read x: %w", errdef.ErrNotFound), false, classMissing},
		{"mismatched digest", fmt.Errorf("failed to read x: %w", content.ErrMismatchedDigest), false, classCorrupt},
		{"trailing data", fmt.Errorf("failed to read x: %w", content.ErrTrailingData), false, classCorrupt},
		{"short read in a layout", fmt.Errorf("failed to read x: %w", io.ErrUnexpectedEOF), true, classCorrupt},
		{"short read from a registry", fmt.Errorf("failed to read x: %w", io.ErrUnexpectedEOF), false, classUnchecked},
		{"invalid body", fmt.Errorf("failed to read x: %w", errCheckInvalid), false, classInvalid},
		{"invalid JSON", syntaxErr, false, classInvalid},
		{"JSON of the wrong type", typeErr, false, classInvalid},
		{"invalid digest", fmt.Errorf("invalid successor descriptor: %w", errdef.ErrInvalidDigest), false, classInvalid},
		{"invalid successor, message only", errors.New("invalid successor descriptor: manifest size must be greater than zero"), false, classInvalid},
		{"network error", errors.New("dial tcp: connection refused"), false, classUnchecked},
		{"HTTP status", errors.New("response status code 403: Forbidden"), true, classUnchecked},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &checker{isLayout: tt.isLayout}
			if got := c.classify(tt.err); got != tt.want {
				t.Errorf("classify(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func Test_checkLayout_resolve(t *testing.T) {
	b := newLayoutBuilder(t)
	manifest, _ := b.image(nil)
	b.tag(manifest, "v1")
	layout, err := newCheckLayout(b.dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{"v1", manifest.Digest.String()} {
		got, err := layout.Resolve(context.Background(), ref)
		if err != nil || got.Digest != manifest.Digest {
			t.Errorf("Resolve(%q) = %v, %v; want %s", ref, got.Digest, err, manifest.Digest)
		}
	}
	if _, err := layout.Resolve(context.Background(), "v2"); !errors.Is(err, errdef.ErrNotFound) {
		t.Errorf("Resolve(unknown tag) error = %v, want not found", err)
	}
}

func Test_checkLayout_resolveNestedDigest(t *testing.T) {
	b := newLayoutBuilder(t)
	m1, _ := b.image(nil, b.blob(testLayerType, []byte("one")))
	m2, _ := b.image(nil, b.blob(testLayerType, []byte("two")))
	idx := b.json(ocispec.MediaTypeImageIndex, ocispec.Index{Versioned: specVersion, Manifests: []ocispec.Descriptor{m1, m2}})
	b.tag(idx, "v1")
	layout, err := newCheckLayout(b.dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []ocispec.Descriptor{m1, m2, idx} {
		got, err := layout.Resolve(context.Background(), want.Digest.String())
		if err != nil || got.Digest != want.Digest || got.MediaType != want.MediaType || got.Size != want.Size {
			t.Errorf("Resolve(%s) = %+v, %v; want %+v", want.Digest, got, err, want)
		}
	}
	unknown := digest.FromString("not in the layout")
	if _, err := layout.Resolve(context.Background(), unknown.String()); !errors.Is(err, errdef.ErrNotFound) {
		t.Errorf("Resolve(unknown digest) error = %v, want not found", err)
	}

	t.Run("check a manifest nested in an index", func(t *testing.T) {
		out, err := runLayoutCheck(t, b.dir, m2.Digest.String(), nil)
		if err != nil || !strings.Contains(out, "Checked 3 nodes: 3 verified, 0 failed") {
			t.Errorf("err = %v, out = %q", err, out)
		}
	})
}

func Test_checkLayout_resolveDuplicateTag(t *testing.T) {
	b := newLayoutBuilder(t)
	first, _ := b.image(nil, b.blob(testLayerType, []byte("first")))
	layer := b.blob(testLayerType, []byte("last"))
	last, _ := b.image(nil, layer)
	b.tag(first, "v1")
	b.tag(last, "v1")

	store, err := oci.NewFromFS(context.Background(), os.DirFS(b.dir))
	if err != nil {
		t.Fatal(err)
	}
	want, err := store.Resolve(context.Background(), "v1")
	if err != nil {
		t.Fatal(err)
	}
	for name, path := range map[string]string{"directory": b.dir, "tar archive": b.tarball()} {
		layout, err := newCheckLayout(path)
		if err != nil {
			t.Fatal(err)
		}
		got, err := layout.Resolve(context.Background(), "v1")
		if err != nil || got.Digest != want.Digest || got.Digest != last.Digest {
			t.Errorf("%s: Resolve(v1) = %s, %v; want %s, as oras-go resolves it", name, got.Digest, err, want.Digest)
		}
	}

	// The artifact that oras-go would use is broken, so the check must fail.
	if err := os.Remove(b.blobPath(layer)); err != nil {
		t.Fatal(err)
	}
	out, err := runLayoutCheck(t, b.dir, "v1", nil)
	wantFailure(t, out, err, "Missing")
}

func Test_checkLayout_resolveImageConfig(t *testing.T) {
	b := newLayoutBuilder(t)
	// An image config has a config field too, but it is not a manifest.
	config := b.json(ocispec.MediaTypeImageConfig, ocispec.Image{Config: ocispec.ImageConfig{Cmd: []string{"sh"}}})
	manifest := b.json(ocispec.MediaTypeImageManifest, ocispec.Manifest{Versioned: specVersion, Config: config, Layers: []ocispec.Descriptor{}})
	b.tag(manifest, "v1")
	out, err := runLayoutCheck(t, b.dir, config.Digest.String(), nil)
	if err != nil || !strings.Contains(out, "Checked 1 nodes: 1 verified, 0 failed") {
		t.Errorf("err = %v, out = %q", err, out)
	}
}

func Test_checkLayout_resolveOversizedManifest(t *testing.T) {
	b := newLayoutBuilder(t)
	// Neither the config nor the layer is stored, so checking only the bytes of
	// the manifest would be a false pass.
	missing := ocispec.Descriptor{MediaType: testLayerType, Digest: digest.FromString("not stored"), Size: 10}
	data, err := json.Marshal(ocispec.Manifest{
		Versioned: specVersion, MediaType: ocispec.MediaTypeImageManifest, Config: missing, Layers: []ocispec.Descriptor{missing},
		Annotations: map[string]string{"padding": strings.Repeat("x", maxCheckManifestSize)},
	})
	if err != nil {
		t.Fatal(err)
	}
	other, _ := b.image(nil)
	b.tag(other, "v1")
	for name, body := range map[string][]byte{"compact": data, "leading whitespace": append([]byte("\n\t "), data...)} {
		t.Run(name, func(t *testing.T) {
			oversized := b.blob(ocispec.MediaTypeImageManifest, body)
			out, err := runLayoutCheck(t, b.dir, oversized.Digest.String(), nil)
			if err == nil || !strings.Contains(err.Error(), "may be a manifest") {
				t.Errorf("err = %v, out = %q; want an error that the blob may be a manifest", err, out)
			}
		})
	}
}

func Test_checkLayout_resolveLargeBlob(t *testing.T) {
	b := newLayoutBuilder(t)
	// A blob larger than maxCheckManifestSize is not a manifest, but its
	// descriptor must still carry the full size.
	big := bytes.Repeat([]byte("x"), maxCheckManifestSize+1234)
	layer := b.blob(testLayerType, big)
	// Tag a manifest that does not reference the layer, so the layer is only
	// reachable by its digest.
	other, _ := b.image(nil, b.blob(testLayerType, []byte("small")))
	b.tag(other, "v1")
	layout, err := newCheckLayout(b.dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := layout.Resolve(context.Background(), layer.Digest.String())
	if err != nil {
		t.Fatal(err)
	}
	if got.Size != int64(len(big)) {
		t.Errorf("Resolve(large blob) size = %d, want %d", got.Size, len(big))
	}
	out, err := runLayoutCheck(t, b.dir, layer.Digest.String(), nil)
	if err != nil || !strings.Contains(out, "Checked 1 nodes: 1 verified, 0 failed") {
		t.Errorf("err = %v, out = %q", err, out)
	}
}

func Test_check_oversizedManifest(t *testing.T) {
	b := newLayoutBuilder(t)
	// The descriptor claims a manifest larger than the limit; it is rejected
	// before any of it is read.
	huge := ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest, Digest: digest.FromString("huge"), Size: maxCheckManifestSize + 1}
	idx := b.json(ocispec.MediaTypeImageIndex, ocispec.Index{Versioned: specVersion, Manifests: []ocispec.Descriptor{huge}})
	b.tag(idx, "v1")
	out, err := runLayoutCheck(t, b.dir, "v1", nil)
	wantFailure(t, out, err, "Unable to check")
}

func Test_isForeignLayer(t *testing.T) {
	for mediaType, want := range map[string]bool{
		"application/vnd.oci.image.layer.nondistributable.v1.tar":      true,
		"application/vnd.oci.image.layer.nondistributable.v1.tar+gzip": true,
		"application/vnd.oci.image.layer.nondistributable.v1.tar+zstd": true,
		"application/vnd.docker.image.rootfs.foreign.diff.tar.gzip":    true,
		"application/vnd.oci.image.layer.v1.tar+gzip":                  false,
		"application/vnd.test.nondistributable.thing":                  false,
		"application/vnd.docker.image.rootfs.diff.tar.gzip":            false,
	} {
		if got := isForeignLayer(mediaType); got != want {
			t.Errorf("isForeignLayer(%q) = %v, want %v", mediaType, got, want)
		}
	}
}

func Test_newCheckLayout_errors(t *testing.T) {
	t.Run("index.json is not JSON", func(t *testing.T) {
		b := newLayoutBuilder(t)
		if err := os.WriteFile(filepath.Join(b.dir, ocispec.ImageIndexFile), []byte("{"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := newCheckLayout(b.dir); err == nil {
			t.Error("a bad index.json was accepted")
		}
	})
	t.Run("a file that is not a tar archive", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "plain.txt")
		if err := os.WriteFile(path, []byte("this is not a tar archive at all"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := newCheckLayout(path); err == nil {
			t.Error("a plain file was accepted as a layout")
		}
	})
	t.Run("a tar archive without index.json", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "empty.tar")
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		tw := tar.NewWriter(f)
		if err := tw.WriteHeader(&tar.Header{Name: "oci-layout", Mode: 0o644, Size: 2, Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte("{}")); err != nil {
			t.Fatal(err)
		}
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := newCheckLayout(path); err == nil || !errors.Is(err, errdef.ErrNotFound) {
			t.Errorf("error = %v, want not found", err)
		}
	})
}

func Test_readTarFile_leadingDotSlash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dot.tar")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	tw := tar.NewWriter(f)
	for _, h := range []tar.Header{
		{Name: "./", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: "./index.json", Typeflag: tar.TypeReg, Mode: 0o644, Size: 2},
	} {
		if err := tw.WriteHeader(&h); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := tw.Write([]byte("{}")); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := readTarFile(path, "index.json")
	if err != nil || string(got) != "{}" {
		t.Errorf("readTarFile() = %q, %v; want {}", got, err)
	}
}

func Test_check_otherManifestFormats(t *testing.T) {
	t.Run("docker manifest and manifest list", func(t *testing.T) {
		b := newLayoutBuilder(t)
		layer := b.blob(docker.MediaTypeConfig+".layer", []byte("docker layer"))
		config := b.blob(docker.MediaTypeConfig, []byte(`{}`))
		manifest := b.json(docker.MediaTypeManifest, ocispec.Manifest{Versioned: specVersion, MediaType: docker.MediaTypeManifest, Config: config, Layers: []ocispec.Descriptor{layer}})
		list := b.json(docker.MediaTypeManifestList, ocispec.Index{Versioned: specVersion, MediaType: docker.MediaTypeManifestList, Manifests: []ocispec.Descriptor{manifest}})
		b.tag(list, "v1")
		if out, err := runLayoutCheck(t, b.dir, "v1", nil); err != nil {
			t.Fatalf("check failed: %v\n%s", err, out)
		}
		b.writeBlob(layer, bytes.Repeat([]byte("x"), int(layer.Size)))
		out, err := runLayoutCheck(t, b.dir, "v1", nil)
		wantFailure(t, out, err, "Corrupt")
	})
	t.Run("artifact manifest with a subject", func(t *testing.T) {
		b := newLayoutBuilder(t)
		subject, _ := b.image(nil)
		blob := b.blob(testLayerType, []byte("artifact blob"))
		artifact := b.json(graph.MediaTypeArtifactManifest, graph.Artifact{
			MediaType: graph.MediaTypeArtifactManifest, ArtifactType: "application/vnd.test", Blobs: []ocispec.Descriptor{blob}, Subject: &subject,
		})
		b.tag(artifact, "v1")
		if out, err := runLayoutCheck(t, b.dir, "v1", nil); err != nil {
			t.Fatalf("check failed: %v\n%s", err, out)
		}
		os.Remove(b.blobPath(blob))
		out, err := runLayoutCheck(t, b.dir, "v1", nil)
		wantFailure(t, out, err, "Missing")
	})
	t.Run("artifact manifest without blobs", func(t *testing.T) {
		b := newLayoutBuilder(t)
		subject, _ := b.image(nil)
		artifact := b.json(graph.MediaTypeArtifactManifest, graph.Artifact{
			MediaType: graph.MediaTypeArtifactManifest, ArtifactType: "application/vnd.test", Subject: &subject,
		})
		b.tag(artifact, "v1")
		out, err := runLayoutCheck(t, b.dir, "v1", nil)
		if err != nil || !strings.Contains(out, "Checked 3 nodes: 3 verified, 0 failed") {
			t.Errorf("err = %v, out = %q", err, out)
		}
	})
	t.Run("an empty index", func(t *testing.T) {
		b := newLayoutBuilder(t)
		idx := b.json(ocispec.MediaTypeImageIndex, ocispec.Index{Versioned: specVersion})
		b.tag(idx, "v1")
		if out, err := runLayoutCheck(t, b.dir, "v1", nil); err != nil {
			t.Fatalf("check failed: %v\n%s", err, out)
		}
	})
	t.Run("a digest reference", func(t *testing.T) {
		b := newLayoutBuilder(t)
		manifest, _ := b.image(nil, b.blob(testLayerType, []byte("by digest")))
		b.tag(manifest, "v1")
		if out, err := runLayoutCheck(t, b.dir, manifest.Digest.String(), nil); err != nil {
			t.Fatalf("check failed: %v\n%s", err, out)
		}
	})
}

// Test_check_manyBlobs checks a wide artifact under the race detector, with a
// mix of intact, missing and corrupt blobs, at several concurrency levels.
func Test_check_manyBlobs(t *testing.T) {
	b := newLayoutBuilder(t)
	var layers []ocispec.Descriptor
	var missing, corrupt int
	for i := 0; i < 300; i++ {
		layer := b.blob(testLayerType, []byte(fmt.Sprintf("layer number %d with some padding to be a little longer", i)))
		layers = append(layers, layer)
		switch i % 10 {
		case 3:
			os.Remove(b.blobPath(layer))
			missing++
		case 7:
			b.writeBlob(layer, bytes.Repeat([]byte("y"), int(layer.Size)))
			corrupt++
		}
	}
	manifest, _ := b.image(nil, layers...)
	b.tag(manifest, "v1")
	layout, err := newCheckLayout(b.dir)
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("Checked 302 nodes: %d verified, %d failed", 302-missing-corrupt, missing+corrupt)
	for _, concurrency := range []int{-5, 0, 1, 3, 64} {
		var out bytes.Buffer
		err := doCheck(context.Background(), layout, "v1", oras.DefaultResolveOptions, true, concurrency, output.NewPrinter(&out, &out))
		if err == nil {
			t.Fatalf("concurrency %d: the check passed", concurrency)
		}
		lines := summary(out.String())
		if len(lines) == 0 || lines[0] != want {
			t.Fatalf("concurrency %d: summary = %q, want first line %q", concurrency, lines, want)
		}
		if got := strings.Count(out.String(), "Missing:"); got != missing {
			t.Errorf("concurrency %d: %d missing lines, want %d", concurrency, got, missing)
		}
		if got := strings.Count(out.String(), "Corrupt:"); got != corrupt {
			t.Errorf("concurrency %d: %d corrupt lines, want %d", concurrency, got, corrupt)
		}
	}
}

func Test_check_largeBlob(t *testing.T) {
	b := newLayoutBuilder(t)
	big := make([]byte, 6<<20)
	for i := range big {
		big[i] = byte(i * 31)
	}
	layer := b.blob(testLayerType, big)
	manifest, _ := b.image(nil, layer)
	b.tag(manifest, "v1")
	if out, err := runLayoutCheck(t, b.dir, "v1", nil); err != nil {
		t.Fatalf("check failed on a large intact blob: %v\n%s", err, out)
	}
	big[len(big)-1] ^= 0xff // one flipped byte at the very end
	b.writeBlob(layer, big)
	out, err := runLayoutCheck(t, b.dir, "v1", nil)
	wantFailure(t, out, err, "Corrupt")
}

// runCheckCmd runs the check command under a parent, as it runs in the CLI:
// error messages name the command by its path.
func runCheckCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "oras", SilenceUsage: true}
	root.AddCommand(checkCmd())
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"check"}, args...))
	err := root.Execute()
	return out.String(), err
}

func Test_checkCmd(t *testing.T) {
	b := newLayoutBuilder(t)
	manifest, _ := b.image(nil, b.blob(testLayerType, []byte("cmd layer")))
	b.tag(manifest, "v1")
	bad := newLayoutBuilder(t)
	badLayer := bad.blob(testLayerType, []byte("bad layer"))
	badManifest, _ := bad.image(nil, badLayer)
	bad.tag(badManifest, "v1")
	os.Remove(bad.blobPath(badLayer))

	t.Run("an intact layout directory", func(t *testing.T) {
		out, err := runCheckCmd(t, "--oci-layout", b.dir+":v1")
		if err != nil || !strings.Contains(out, "Checked 3 nodes: 3 verified, 0 failed") {
			t.Errorf("err = %v, out = %q", err, out)
		}
	})
	t.Run("a line for each node, which has no title", func(t *testing.T) {
		out, err := runCheckCmd(t, "--oci-layout", b.dir+":v1")
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Count(out, "Verified "); got != 3 {
			t.Errorf("got %d Verified lines, want 3: %q", got, out)
		}
	})
	t.Run("an intact layout archive", func(t *testing.T) {
		if out, err := runCheckCmd(t, "--oci-layout", b.tarball()+":v1"); err != nil {
			t.Errorf("err = %v, out = %q", err, out)
		}
	})
	t.Run("the concurrency flag", func(t *testing.T) {
		for _, n := range []string{"1", "0", "-1"} {
			if out, err := runCheckCmd(t, "--concurrency", n, "--oci-layout", b.dir+":v1"); err != nil {
				t.Errorf("--concurrency %s: err = %v, out = %q", n, err, out)
			}
		}
	})
	t.Run("a failing layout", func(t *testing.T) {
		out, err := runCheckCmd(t, "--oci-layout", bad.dir+":v1")
		if err == nil || !strings.Contains(err.Error(), "check failed: 1 of 3 nodes did not pass") {
			t.Errorf("err = %v", err)
		}
		if !strings.Contains(out, "Missing:") {
			t.Errorf("out = %q, want a Missing line", out)
		}
	})
	t.Run("whole layout directory via --oci-layout", func(t *testing.T) {
		out, err := runCheckCmd(t, "--oci-layout", b.dir)
		if err != nil || !strings.Contains(out, "Checked 3 nodes: 3 verified, 0 failed") {
			t.Errorf("err = %v, out = %q", err, out)
		}
	})
	t.Run("whole layout directory via --oci-layout-path", func(t *testing.T) {
		out, err := runCheckCmd(t, "--oci-layout-path", b.dir)
		if err != nil || !strings.Contains(out, "Checked 3 nodes: 3 verified, 0 failed") {
			t.Errorf("err = %v, out = %q", err, out)
		}
	})
	t.Run("whole layout archive via --oci-layout", func(t *testing.T) {
		out, err := runCheckCmd(t, "--oci-layout", b.tarball())
		if err != nil || !strings.Contains(out, "Checked 3 nodes: 3 verified, 0 failed") || !strings.Contains(out, "Not checked: unreferenced blobs and stray files") {
			t.Errorf("err = %v, out = %q", err, out)
		}
	})
	t.Run("whole layout archive via --oci-layout-path", func(t *testing.T) {
		out, err := runCheckCmd(t, "--oci-layout-path", b.tarball())
		if err != nil || !strings.Contains(out, "Checked 3 nodes: 3 verified, 0 failed") || !strings.Contains(out, "Not checked: unreferenced blobs and stray files") {
			t.Errorf("err = %v, out = %q", err, out)
		}
	})
	t.Run("whole layout with --platform fails", func(t *testing.T) {
		if _, err := runCheckCmd(t, "--platform", "linux/arm64", "--oci-layout", b.dir); err == nil || !strings.Contains(err.Error(), "--platform") {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("no reference for registry", func(t *testing.T) {
		if _, err := runCheckCmd(t, "localhost:5000/hello-world"); err == nil || !strings.Contains(err.Error(), "no tag or digest specified") {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("trailing colon without reference", func(t *testing.T) {
		if _, err := runCheckCmd(t, "--oci-layout", b.dir+":"); err == nil || !strings.Contains(err.Error(), "no tag or digest specified") {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("no argument", func(t *testing.T) {
		if _, err := runCheckCmd(t); err == nil {
			t.Error("a missing argument was accepted")
		}
	})
	t.Run("two arguments", func(t *testing.T) {
		if _, err := runCheckCmd(t, "--oci-layout", b.dir+":v1", "other"); err == nil {
			t.Error("two arguments were accepted")
		}
	})
	t.Run("two arguments with --oci-layout-path", func(t *testing.T) {
		if _, err := runCheckCmd(t, "--oci-layout-path", b.dir, "v1", "other"); err == nil {
			t.Error("two arguments with --oci-layout-path were accepted")
		}
	})
	t.Run("a path that does not exist", func(t *testing.T) {
		if _, err := runCheckCmd(t, "--oci-layout", filepath.Join(t.TempDir(), "nope")+":v1"); err == nil || !strings.Contains(err.Error(), "invalid argument") {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("an unknown tag", func(t *testing.T) {
		if _, err := runCheckCmd(t, "--oci-layout", b.dir+":nope"); err == nil || !strings.Contains(err.Error(), "failed to resolve") {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("the help text", func(t *testing.T) {
		out, err := runCheckCmd(t, "--help")
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"[Experimental]", "does not detect", "Every byte", "--concurrency", "--platform", "--oci-layout"} {
			if !strings.Contains(out, want) {
				t.Errorf("help text is missing %q", want)
			}
		}
	})
}

func Test_checkCmd_registry(t *testing.T) {
	layerData := []byte("registry layer for the command")
	configData := []byte(`{}`)
	layer := ocispec.Descriptor{MediaType: testLayerType, Digest: digest.FromBytes(layerData), Size: int64(len(layerData))}
	config := ocispec.Descriptor{MediaType: "application/vnd.test.config", Digest: digest.FromBytes(configData), Size: int64(len(configData))}
	manifestData, err := json.Marshal(ocispec.Manifest{Versioned: specVersion, MediaType: ocispec.MediaTypeImageManifest, Config: config, Layers: []ocispec.Descriptor{layer}})
	if err != nil {
		t.Fatal(err)
	}
	manifest := ocispec.Descriptor{MediaType: ocispec.MediaTypeImageManifest, Digest: digest.FromBytes(manifestData), Size: int64(len(manifestData))}
	bodies := map[digest.Digest][]byte{layer.Digest: layerData, config.Digest: configData, manifest.Digest: manifestData}

	t.Run("an intact artifact by tag and by digest", func(t *testing.T) {
		server := httptest.NewServer(&fakeRegistry{manifest: manifest, body: bodies})
		defer server.Close()
		host := strings.TrimPrefix(server.URL, "http://")
		for _, ref := range []string{"v1", "@" + manifest.Digest.String()} {
			sep := ":"
			if strings.HasPrefix(ref, "@") {
				sep = ""
			}
			out, err := runCheckCmd(t, "--plain-http", host+"/test/artifact"+sep+ref)
			if err != nil || !strings.Contains(out, "Checked 3 nodes: 3 verified, 0 failed") {
				t.Errorf("%s: err = %v, out = %q", ref, err, out)
			}
		}
	})
	t.Run("a blob the registry refuses", func(t *testing.T) {
		reg := &fakeRegistry{manifest: manifest, body: bodies, blob: func(w http.ResponseWriter, _ *http.Request, d digest.Digest) {
			if d == layer.Digest {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			_, _ = w.Write(bodies[d])
		}}
		server := httptest.NewServer(reg)
		defer server.Close()
		out, err := runCheckCmd(t, "--plain-http", strings.TrimPrefix(server.URL, "http://")+"/test/artifact:v1")
		if err == nil || !strings.Contains(out, "Unable to check:") {
			t.Errorf("err = %v, out = %q", err, out)
		}
		var respErr interface{ HTTPStatusCode() int }
		if errors.As(err, &respErr) {
			t.Errorf("the summary error wraps a registry response, so it would be shown as a registry error: %v", err)
		}
	})
	t.Run("a tag the registry does not have", func(t *testing.T) {
		server := httptest.NewServer(&fakeRegistry{manifest: manifest, body: bodies})
		defer server.Close()
		_, err := runCheckCmd(t, "--plain-http", strings.TrimPrefix(server.URL, "http://")+"/test/artifact:nope")
		if err == nil || !strings.Contains(err.Error(), "failed to resolve") {
			t.Errorf("err = %v", err)
		}
	})
}

func Test_checkWholeLayout_directory(t *testing.T) {
	t.Run("intact layout with multiple tags and a shared layer", func(t *testing.T) {
		b := newLayoutBuilder(t)
		shared := b.blob(testLayerType, []byte("shared layer"))
		extra := b.blob(testLayerType, []byte("second layer"))
		m1, _ := b.image(nil, shared)
		m2, _ := b.image(nil, shared, extra)
		b.tag(m1, "v1")
		b.tag(m2, "v2")

		out, err := runWholeLayoutCheck(t, b.dir)
		if err != nil {
			t.Fatalf("unexpected error: %v\noutput:\n%s", err, out)
		}
		// 2 manifests + 1 shared config + 2 layers = 5 unique blobs
		lines := summary(out)
		if len(lines) != 1 || lines[0] != "Checked 5 nodes: 5 verified, 0 failed" {
			t.Errorf("summary = %q, want 5 verified, 0 failed", lines)
		}
	})

	t.Run("manifest with wrong layer size in descriptor", func(t *testing.T) {
		b := newLayoutBuilder(t)
		layer := b.blob(testLayerType, []byte("real content"))
		wrongSizeLayer := layer
		wrongSizeLayer.Size += 10
		m, _ := b.image(nil, wrongSizeLayer)
		b.tag(m, "v1")

		out, err := runWholeLayoutCheck(t, b.dir)
		if err == nil {
			t.Fatalf("expected error, got nil\noutput:\n%s", out)
		}
		lines := summary(out)
		if len(lines) != 2 || lines[0] != "Checked 3 nodes: 2 verified, 1 failed" || !strings.HasPrefix(lines[1], "Corrupt: "+testLayerType+" "+layer.Digest.String()+":") {
			t.Errorf("summary = %q, err = %v", lines, err)
		}
	})

	t.Run("index.json types an image manifest as an index with a missing layer", func(t *testing.T) {
		b := newLayoutBuilder(t)
		layer := b.blob(testLayerType, []byte("gone"))
		m, _ := b.image(nil, layer)
		if err := os.Remove(b.blobPath(layer)); err != nil {
			t.Fatal(err)
		}
		m.MediaType = ocispec.MediaTypeImageIndex
		b.tag(m, "v1")

		out, err := runWholeLayoutCheck(t, b.dir)
		if err == nil {
			t.Fatalf("expected error, got nil\noutput:\n%s", out)
		}
		if !strings.Contains(out, "Invalid: "+ocispec.MediaTypeImageIndex+" "+m.Digest.String()+":") {
			t.Errorf("want Invalid line for mistyped manifest, got:\n%s", out)
		}
	})

	t.Run("corrupt index.json still checks blobs", func(t *testing.T) {
		b := newLayoutBuilder(t)
		layer := b.blob(testLayerType, []byte("layer data"))
		b.writeBlob(layer, []byte("tampered!!"))
		if err := os.WriteFile(filepath.Join(b.dir, ocispec.ImageIndexFile), []byte("{not json"), 0o644); err != nil {
			t.Fatal(err)
		}

		out, err := runWholeLayoutCheck(t, b.dir)
		if err == nil {
			t.Fatalf("expected error, got nil\noutput:\n%s", out)
		}
		if !strings.Contains(out, "Invalid: index.json:") {
			t.Errorf("want Invalid: index.json line, got:\n%s", out)
		}
		if !strings.Contains(out, "Corrupt: "+layer.Digest.String()+":") {
			t.Errorf("want Corrupt line for tampered blob, got:\n%s", out)
		}
	})

	t.Run("missing index.json and corrupt oci-layout", func(t *testing.T) {
		b := newLayoutBuilder(t)
		b.blob(testLayerType, []byte("blob"))
		if err := os.WriteFile(filepath.Join(b.dir, ocispec.ImageLayoutFile), []byte(`{"imageLayoutVersion":"2.0.0"}`), 0o644); err != nil {
			t.Fatal(err)
		}

		out, err := runWholeLayoutCheck(t, b.dir)
		if err == nil {
			t.Fatalf("expected error, got nil\noutput:\n%s", out)
		}
		if !strings.Contains(out, "Checked 0 nodes: 0 verified, 0 failed") {
			t.Errorf("want Checked 0 nodes: 0 verified, 0 failed, got:\n%s", out)
		}
		if !strings.Contains(out, "Layout problems: 2") {
			t.Errorf("want Layout problems: 2, got:\n%s", out)
		}
		if !strings.Contains(out, "Invalid: index.json:") {
			t.Errorf("want Invalid: index.json line, got:\n%s", out)
		}
		if !strings.Contains(out, "Invalid: oci-layout:") {
			t.Errorf("want Invalid: oci-layout line, got:\n%s", out)
		}
	})

	t.Run("interrupted write leftover in ingest is informational", func(t *testing.T) {
		b := newLayoutBuilder(t)
		m, _ := b.image(nil, b.blob(testLayerType, []byte("ok")))
		b.tag(m, "v1")
		ingestDir := filepath.Join(b.dir, "ingest")
		if err := os.MkdirAll(ingestDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(ingestDir, "partial"), []byte("half written"), 0o644); err != nil {
			t.Fatal(err)
		}

		out, err := runWholeLayoutCheck(t, b.dir)
		if err != nil {
			t.Fatalf("leftover should be informational, got err = %v\noutput:\n%s", err, out)
		}
		if !strings.Contains(out, "Checked 3 nodes: 3 verified, 0 failed") {
			t.Errorf("unexpected summary:\n%s", out)
		}
		if !strings.Contains(out, "Note: ingest") || !strings.Contains(out, "leftover from an interrupted write") {
			t.Errorf("want leftover Note line, got:\n%s", out)
		}
	})

	t.Run("foreign layer absent passes with note", func(t *testing.T) {
		b := newLayoutBuilder(t)
		local := b.blob(testLayerType, []byte("local"))
		foreign := ocispec.Descriptor{
			MediaType: "application/vnd.oci.image.layer.nondistributable.v1.tar",
			Digest:    digest.FromString("remote foreign layer"),
			Size:      42,
			URLs:      []string{"https://example.com/layer"},
		}
		m, _ := b.image(nil, local, foreign)
		b.tag(m, "v1")

		out, err := runWholeLayoutCheck(t, b.dir)
		if err != nil {
			t.Fatalf("absent foreign layer should pass, got err = %v\noutput:\n%s", err, out)
		}
		lines := summary(out)
		if len(lines) != 2 || lines[0] != "Checked 3 nodes: 3 verified, 0 failed" || !strings.Contains(lines[1], "Not checked: 1 foreign layers") {
			t.Errorf("summary = %q", lines)
		}
	})

	t.Run("corrupt referrer manifest is reported once", func(t *testing.T) {
		b := newLayoutBuilder(t)
		subj, _ := b.image(nil, b.blob(testLayerType, []byte("subject layer")))
		b.tag(subj, "v1")
		ref, _ := b.image(&subj, b.blob(testLayerType, []byte("sig layer")))
		b.addIndex(ref)
		// Corrupt the referrer manifest bytes (same length so it's a digest mismatch).
		corrupt := bytes.Repeat([]byte("x"), int(ref.Size))
		b.writeBlob(ref, corrupt)

		out, err := runWholeLayoutCheck(t, b.dir)
		if err == nil {
			t.Fatalf("expected error, got nil\noutput:\n%s", out)
		}
		lines := summary(out)
		corruptCount := 0
		for _, line := range lines {
			if strings.Contains(line, ref.Digest.String()) {
				corruptCount++
			}
			if strings.HasPrefix(line, "Unable to check:") {
				t.Errorf("corrupt referrer should not also be reported as Unable to check: %q", line)
			}
		}
		if corruptCount != 1 {
			t.Errorf("expected referrer digest once in summary, got %d times in %q", corruptCount, lines)
		}
		if !strings.Contains(out, "Corrupt: "+ocispec.MediaTypeImageManifest+" "+ref.Digest.String()+":") {
			t.Errorf("want Corrupt line for referrer, got:\n%s", out)
		}
	})

	t.Run("referrer listed in index.json but missing fails", func(t *testing.T) {
		b := newLayoutBuilder(t)
		subj, _ := b.image(nil, b.blob(testLayerType, []byte("subject")))
		b.tag(subj, "v1")
		missingRef := ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageManifest,
			Digest:    digest.FromString("missing referrer"),
			Size:      100,
		}
		b.addIndex(missingRef)

		out, err := runWholeLayoutCheck(t, b.dir)
		if err == nil {
			t.Fatalf("expected error, got nil\noutput:\n%s", out)
		}
		if !strings.Contains(out, "Missing: "+ocispec.MediaTypeImageManifest+" "+missingRef.Digest.String()+":") {
			t.Errorf("want Missing line for missing referrer in index.json, got:\n%s", out)
		}
	})

	t.Run("absent subject not in index.json is a warning", func(t *testing.T) {
		b := newLayoutBuilder(t)
		absentSubj := ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageManifest,
			Digest:    digest.FromString("absent subject"),
			Size:      55,
		}
		ref, _ := b.image(&absentSubj, b.blob(testLayerType, []byte("sig")))
		b.tag(ref, "sig")

		out, err := runWholeLayoutCheck(t, b.dir)
		if err != nil {
			t.Fatalf("absent subject should warn and pass, got err = %v\noutput:\n%s", err, out)
		}
		if !strings.Contains(out, "Warning: the subject "+absentSubj.Digest.String()) {
			t.Errorf("want Warning line for absent subject, got:\n%s", out)
		}
	})

	t.Run("layout with no tags reports unreferenced blobs as info", func(t *testing.T) {
		b := newLayoutBuilder(t)
		b.blob(testLayerType, []byte("orphan 1"))
		b.blob(testLayerType, []byte("orphan 2"))
		b.writeIndex() // empty manifests array

		out, err := runWholeLayoutCheck(t, b.dir)
		if err != nil {
			t.Fatalf("unreferenced intact blobs should pass, got err = %v\noutput:\n%s", err, out)
		}
		lines := summary(out)
		if len(lines) != 2 || lines[0] != "Checked 0 nodes: 0 verified, 0 failed" || !strings.Contains(lines[1], "Note: 2 unreferenced blobs") {
			t.Errorf("summary = %q", lines)
		}
	})

	t.Run("corrupt unreferenced blob fails", func(t *testing.T) {
		b := newLayoutBuilder(t)
		orphan := b.blob(testLayerType, []byte("orphan"))
		b.writeBlob(orphan, []byte("broken"))
		b.writeIndex()

		out, err := runWholeLayoutCheck(t, b.dir)
		if err == nil {
			t.Fatalf("expected error, got nil\noutput:\n%s", out)
		}
		if !strings.Contains(out, "Corrupt: "+orphan.Digest.String()+":") {
			t.Errorf("want Corrupt line for unreferenced blob, got:\n%s", out)
		}
	})

	t.Run("stray file and unsupported algorithm in blobs are unable to check", func(t *testing.T) {
		b := newLayoutBuilder(t)
		m, _ := b.image(nil, b.blob(testLayerType, []byte("ok")))
		b.tag(m, "v1")
		if err := os.WriteFile(filepath.Join(b.dir, "blobs", "stray.txt"), []byte("oops"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(b.dir, "blobs", "md5"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(b.dir, "blobs", "md5", "d41d8cd98f00b204e9800998ecf8427e"), nil, 0o644); err != nil {
			t.Fatal(err)
		}

		out, err := runWholeLayoutCheck(t, b.dir)
		if err == nil {
			t.Fatalf("expected error, got nil\noutput:\n%s", out)
		}
		if !strings.Contains(err.Error(), "2 layout problems") {
			t.Errorf("err = %v, want 2 layout problems", err)
		}
		lines := summary(out)
		if len(lines) < 2 || lines[0] != "Checked 3 nodes: 3 verified, 0 failed" || lines[1] != "Layout problems: 2" {
			t.Errorf("summary = %q, want 3 verified/0 failed and 2 layout problems", lines)
		}
		if !strings.Contains(out, "Unable to check: blobs/stray.txt:") {
			t.Errorf("want Unable to check for stray file, got:\n%s", out)
		}
		if !strings.Contains(out, "Unable to check: blobs/md5") {
			t.Errorf("want Unable to check for unsupported alg, got:\n%s", out)
		}
	})

	t.Run("wrong path or directory without blobs", func(t *testing.T) {
		if _, err := runWholeLayoutCheck(t, filepath.Join(t.TempDir(), "does-not-exist")); err == nil || !strings.Contains(err.Error(), "invalid argument") {
			t.Errorf("missing path err = %v", err)
		}
		emptyDir := t.TempDir()
		if _, err := runWholeLayoutCheck(t, emptyDir); err == nil || !strings.Contains(err.Error(), "not an OCI image layout") {
			t.Errorf("dir without blobs err = %v", err)
		}
	})

	t.Run("cancellation stops without summary", func(t *testing.T) {
		b := newLayoutBuilder(t)
		m, _ := b.image(nil, b.blob(testLayerType, []byte("ok")))
		b.tag(m, "v1")

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		var out bytes.Buffer
		err := doCheckWholeLayout(ctx, b.dir, 1, output.NewPrinter(&out, &out))
		if !errors.Is(err, context.Canceled) {
			t.Errorf("err = %v, want context.Canceled", err)
		}
		if len(summary(out.String())) != 0 {
			t.Errorf("cancelled check should print no summary, got:\n%s", out.String())
		}
	})
}

func Test_checkWholeLayout_tar(t *testing.T) {
	t.Run("intact multi-tag tar archive with shared layer", func(t *testing.T) {
		b := newLayoutBuilder(t)
		shared := b.blob(testLayerType, []byte("shared"))
		m1, _ := b.image(nil, shared)
		m2, _ := b.image(nil, shared, b.blob(testLayerType, []byte("extra")))
		b.tag(m1, "v1")
		b.tag(m2, "v2")

		out, err := runWholeLayoutCheck(t, b.tarball())
		if err != nil {
			t.Fatalf("unexpected error: %v\noutput:\n%s", err, out)
		}
		lines := summary(out)
		if len(lines) != 2 || lines[0] != "Checked 5 nodes: 5 verified, 0 failed" || !strings.Contains(lines[1], "Not checked: unreferenced blobs and stray files are not checked in a tar archive") {
			t.Errorf("summary = %q", lines)
		}
	})

	t.Run("corrupt layer in tar archive", func(t *testing.T) {
		b := newLayoutBuilder(t)
		layer := b.blob(testLayerType, []byte("good"))
		m, _ := b.image(nil, layer)
		b.tag(m, "v1")
		b.writeBlob(layer, []byte("evil"))

		out, err := runWholeLayoutCheck(t, b.tarball())
		if err == nil || !strings.Contains(out, "Corrupt: "+testLayerType+" "+layer.Digest.String()+":") {
			t.Errorf("err = %v, out = %q", err, out)
		}
	})

	t.Run("missing layer in tar archive", func(t *testing.T) {
		b := newLayoutBuilder(t)
		layer := b.blob(testLayerType, []byte("gone"))
		m, _ := b.image(nil, layer)
		b.tag(m, "v1")
		if err := os.Remove(b.blobPath(layer)); err != nil {
			t.Fatal(err)
		}

		out, err := runWholeLayoutCheck(t, b.tarball())
		if err == nil || !strings.Contains(out, "Missing: "+testLayerType+" "+layer.Digest.String()+":") {
			t.Errorf("err = %v, out = %q", err, out)
		}
	})

	t.Run("wrong layer size in tar archive", func(t *testing.T) {
		b := newLayoutBuilder(t)
		layer := b.blob(testLayerType, []byte("real content"))
		wrong := layer
		wrong.Size -= 3
		m, _ := b.image(nil, wrong)
		b.tag(m, "v1")

		out, err := runWholeLayoutCheck(t, b.tarball())
		if err == nil || !strings.Contains(out, "Corrupt: "+testLayerType+" "+layer.Digest.String()+":") {
			t.Errorf("err = %v, out = %q", err, out)
		}
	})

	t.Run("foreign layer absent in tar archive passes", func(t *testing.T) {
		b := newLayoutBuilder(t)
		foreign := ocispec.Descriptor{
			MediaType: "application/vnd.oci.image.layer.nondistributable.v1.tar",
			Digest:    digest.FromString("remote"),
			Size:      10,
			URLs:      []string{"https://example.com/layer"},
		}
		m, _ := b.image(nil, b.blob(testLayerType, []byte("ok")), foreign)
		b.tag(m, "v1")

		out, err := runWholeLayoutCheck(t, b.tarball())
		if err != nil {
			t.Fatalf("unexpected error: %v\noutput:\n%s", err, out)
		}
		if !strings.Contains(out, "Not checked: 1 foreign layers") || !strings.Contains(out, "Not checked: unreferenced blobs and stray files") {
			t.Errorf("unexpected output:\n%s", out)
		}
	})

	t.Run("missing oci-layout in tar archive", func(t *testing.T) {
		b := newLayoutBuilder(t)
		m, _ := b.image(nil, b.blob(testLayerType, []byte("ok")))
		b.tag(m, "v1")
		if err := os.Remove(filepath.Join(b.dir, ocispec.ImageLayoutFile)); err != nil {
			t.Fatal(err)
		}

		out, err := runWholeLayoutCheck(t, b.tarball())
		if err == nil {
			t.Fatalf("expected error, got nil\noutput:\n%s", out)
		}
		if !strings.Contains(out, "Checked 3 nodes: 3 verified, 0 failed") {
			t.Errorf("want Checked 3 nodes: 3 verified, 0 failed, got:\n%s", out)
		}
		if !strings.Contains(out, "Layout problems: 1") {
			t.Errorf("want Layout problems: 1, got:\n%s", out)
		}
		if !strings.Contains(out, "Invalid: oci-layout:") {
			t.Errorf("want Invalid: oci-layout line, got:\n%s", out)
		}
	})

	t.Run("invalid oci-layout version in tar archive", func(t *testing.T) {
		b := newLayoutBuilder(t)
		m, _ := b.image(nil, b.blob(testLayerType, []byte("ok")))
		b.tag(m, "v1")
		if err := os.WriteFile(filepath.Join(b.dir, ocispec.ImageLayoutFile), []byte(`{"imageLayoutVersion":"2.0.0"}`), 0o644); err != nil {
			t.Fatal(err)
		}

		out, err := runWholeLayoutCheck(t, b.tarball())
		if err == nil {
			t.Fatalf("expected error, got nil\noutput:\n%s", out)
		}
		if !strings.Contains(out, "Checked 3 nodes: 3 verified, 0 failed") {
			t.Errorf("want Checked 3 nodes: 3 verified, 0 failed, got:\n%s", out)
		}
		if !strings.Contains(out, "Layout problems: 1") {
			t.Errorf("want Layout problems: 1, got:\n%s", out)
		}
		if !strings.Contains(out, "Invalid: oci-layout:") {
			t.Errorf("want Invalid: oci-layout line, got:\n%s", out)
		}
	})

	t.Run("not a tar archive", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "not-a-tar")
		if err := os.WriteFile(p, []byte("plain text"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := runWholeLayoutCheck(t, p)
		if err == nil || strings.Count(err.Error(), "not an OCI image layout") != 1 {
			t.Errorf("err = %v, want single 'not an OCI image layout' prefix", err)
		}
	})
}

type countingStorage struct {
	content.ReadOnlyStorage
	fetches map[digest.Digest]*atomic.Int32
}

func (s *countingStorage) Fetch(ctx context.Context, target ocispec.Descriptor) (io.ReadCloser, error) {
	if c, ok := s.fetches[target.Digest]; ok {
		c.Add(1)
	}
	return s.ReadOnlyStorage.Fetch(ctx, target)
}

func Test_checkWholeLayout_verifiedSetSkipsRereadingLayers(t *testing.T) {
	b := newLayoutBuilder(t)
	layer := b.blob(testLayerType, []byte("large layer"))
	m1, _ := b.image(nil, layer)
	wrongLayer := layer
	wrongLayer.Size += 5
	m2, _ := b.image(nil, wrongLayer)

	storage := oci.NewStorageFromFS(os.DirFS(b.dir))
	var layerReads atomic.Int32
	counted := &countingStorage{
		ReadOnlyStorage: storage,
		fetches:         map[digest.Digest]*atomic.Int32{layer.Digest: &layerReads},
	}

	// Pre-seed VerifiedSet with the real on-disk size, as doCheckLayoutDir does.
	verifiedSet := oras.NewVerifiedSet()
	verifiedSet.Add(ocispec.Descriptor{Digest: layer.Digest, Size: layer.Size})

	var out bytes.Buffer
	c := &checker{
		isLayout:  true,
		printer:   output.NewPrinter(&out, &out),
		bySubject: make(map[digest.Digest]bool),
		byOther:   make(map[digest.Digest]bool),
		foreign:   make(map[digest.Digest]bool),
	}

	// Walking m1 (matching size) must not re-read the layer at all.
	res1, err := c.walkLayoutManifests(context.Background(), counted, []ocispec.Descriptor{m1}, verifiedSet, 2)
	if err != nil || len(res1.failures) != 0 {
		t.Fatalf("m1 walk failed: err=%v, failures=%v", err, res1.failures)
	}
	if got := layerReads.Load(); got != 0 {
		t.Errorf("layer was read %d times despite VerifiedSet, want 0", got)
	}

	// Walking m2 (wrong descriptor size) must read the layer and flag Corrupt.
	res2, err := c.walkLayoutManifests(context.Background(), counted, []ocispec.Descriptor{m2}, verifiedSet, 2)
	if err != nil {
		t.Fatalf("m2 walk returned unexpected error: %v", err)
	}
	if got := layerReads.Load(); got != 1 {
		t.Errorf("wrong-size layer was read %d times, want 1", got)
	}
	if len(res2.failures) != 1 || res2.failures[0].class != classCorrupt {
		t.Errorf("m2 failures = %+v, want 1 Corrupt failure", res2.failures)
	}
}

func Test_blobPathDigest(t *testing.T) {
	validDigest := digest.FromString("hello")
	tests := []struct {
		path   string
		wantOK bool
		want   digest.Digest
	}{
		{"blobs/sha256/" + validDigest.Encoded(), true, validDigest},
		{"blobs/stray.txt", false, ""},
		{"blobs/sha256/not-hex", false, ""},
		{"blobs/sha256/" + validDigest.Encoded() + "/extra", false, ""},
		{"blobs/md5/d41d8cd98f00b204e9800998ecf8427e", false, ""},
		{"index.json", false, ""},
	}
	for _, tt := range tests {
		got, ok := blobPathDigest(tt.path)
		if ok != tt.wantOK || got != tt.want {
			t.Errorf("blobPathDigest(%q) = (%q, %v), want (%q, %v)", tt.path, got, ok, tt.want, tt.wantOK)
		}
	}
}
