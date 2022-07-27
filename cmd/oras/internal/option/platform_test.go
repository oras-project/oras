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

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/pflag"
	"oras.land/oras-go/v2/registry/remote"
)

func TestPlatform_ApplyFlags(t *testing.T) {
	var test struct{ Platform }
	ApplyFlags(&test, pflag.NewFlagSet("oras-test", pflag.ExitOnError))
	if test.Platform.Platform != "" {
		t.Fatalf("expecting platform to be empty but got: %v", test.Platform.Platform)
	}
}

func TestPlatform_parse_invalidPlatform(t *testing.T) {
	var checker = func(flag string) {
		if _, err := (&Platform{flag}).parse(); err == nil {
			t.Fatalf("expecting parse error for flag: %q", flag)
		}
	}

	checker("")
	checker("os/")
	checker("os")
	checker("/arch")
	checker("/arch/variant")
	checker("os/arch/variant/llama")
}

func TestPlatform_parse(t *testing.T) {
	var checker = func(flag string, want ocispec.Platform) {
		got, err := (&Platform{flag}).parse()
		if err != nil {
			t.Fatalf("unexpected parse error for flag: %q", flag)
		}
		if got.OS != want.OS || got.Architecture != want.Architecture || got.Variant != want.Variant || got.OSVersion != want.OSVersion {
			t.Fatalf("Parse result unmatched: expecting %v, got %v", want, got)
		}
	}

	checker("os/aRcH", ocispec.Platform{OS: "os", Architecture: "aRcH"})
	checker("os/aRcH/", ocispec.Platform{OS: "os", Architecture: "aRcH", Variant: ""})
	checker("os/aRcH/vAriAnt", ocispec.Platform{OS: "os", Architecture: "aRcH", Variant: "vAriAnt"})
	checker("os/aRcH/vAriAnt:osversion", ocispec.Platform{OS: "os", Architecture: "aRcH", Variant: "vAriAnt", OSVersion: "osversion"})
	checker("os/aRcH/vAriAnt:os::::version", ocispec.Platform{OS: "os", Architecture: "aRcH", Variant: "vAriAnt", OSVersion: "os::::version"})
}

type entry struct {
	ocispec.Descriptor
	content []byte
}
type repoMock struct {
	content map[string]entry
	remote.Repository
}

func (mock *repoMock) FetchReference(ctx context.Context, ref string) (ocispec.Descriptor, io.ReadCloser, error) {
	if r, ok := mock.content[ref]; ok {
		return r.Descriptor, io.NopCloser(bytes.NewReader(r.content)), nil
	}
	return ocispec.Descriptor{}, nil, fmt.Errorf("Got unexpected reference %q", ref)
}

func (mock *repoMock) Fetch(ctx context.Context, target ocispec.Descriptor) (io.ReadCloser, error) {
	if r, ok := mock.content[target.Digest.String()]; ok {
		return io.NopCloser(bytes.NewReader(r.content)), nil
	}
	return nil, fmt.Errorf("Unexpected descriptor %v", target)
}

func newRepoMock(contents []string, types []string) *repoMock {
	var mock repoMock
	if len(contents) != len(types) {
		panic(fmt.Sprintf("Invalid mocking data: %v, %v", contents, types))
	}

	mock.content = make(map[string]entry)
	for i, c := range contents {
		b := []byte(c)
		desc := ocispec.Descriptor{
			MediaType: types[i],
			Digest:    digest.FromBytes(b),
			Size:      int64(len(b)),
		}
		mock.content[string(desc.Digest)] = entry{desc, b}
	}
	return &mock
}

const (
	index = `{"manifests":[{"digest":"sha256:baf0239e48ff4c47ebac3ba02b5cf1506b69cd5a0c0d0c825a53ba65976fb942","mediaType":"application\/vnd.docker.distribution.manifest.v2+json","platform":{"architecture":"amd64","os":"linux"},"size":11},{"digest":"sha256:27cb13102d774dc36e0bc93f528db7e4f004a6e9636cb6926b1e389668535309","mediaType":"application\/vnd.docker.distribution.manifest.v2+json","platform":{"architecture":"arm","os":"linux","variant":"v5"},"size":12}]}`
	amd64 = "linux/amd64"
	armv5 = "linux/arm/v5"
	armv7 = "linux/arm/v7"
)

func TestPlatform_FetchManifest_indexAndPlatformMatch(t *testing.T) {
	repo := newRepoMock(
		[]string{index, amd64, armv5},
		[]string{ocispec.MediaTypeImageIndex, ocispec.MediaTypeImageManifest, ocispec.MediaTypeImageManifest})

	// Get index manifest
	opts := Platform{""}
	indexBytes := []byte(index)
	got, err := opts.FetchManifest(context.Background(), repo, digest.FromBytes(indexBytes).String())
	if err != nil || !bytes.Equal(got, indexBytes) {
		t.Fatal(err)
	}

	// Get manifest for specific platform
	opts = Platform{amd64}
	want := []byte(amd64)
	got, err = opts.FetchManifest(context.Background(), repo, digest.FromBytes(indexBytes).String())
	if err != nil || !bytes.Equal(got, want) {
		t.Fatal(err)
	}
	opts = Platform{armv5}
	want = []byte(armv5)
	got, err = opts.FetchManifest(context.Background(), repo, digest.FromBytes(indexBytes).String())
	if err != nil || !bytes.Equal(got, want) {
		t.Fatal(err)
	}
}

func TestPlatform_FetchManifest_errNotMulti(t *testing.T) {
	repo := newRepoMock(
		[]string{amd64},
		[]string{ocispec.MediaTypeImageManifest})

	// Unknow media type
	opts := Platform{amd64}
	_, err := opts.FetchManifest(context.Background(), repo, digest.FromBytes([]byte(amd64)).String())
	if !errors.Is(err, errMediatypeUnsupported) {
		t.Fatalf("Expecting error: %v, got: %v", errMediatypeUnsupported, err)
	}
}
func TestPlatform_FetchManifest_errNoMatch(t *testing.T) {
	// No matched platform found
	opts := Platform{armv7}
	_, err := opts.FetchManifest(
		context.Background(),
		newRepoMock([]string{index}, []string{ocispec.MediaTypeImageIndex}),
		digest.FromBytes([]byte(index)).String())
	if !errors.Is(err, errNoMatchFound) {
		t.Fatalf("Expecting error: %v, got: %v", errNoMatchFound, err)
	}
}

func TestPlatform_FetchManifest_miscErr(t *testing.T) {
	// Should throw err when repo is empty
	opts := Platform{""}
	ret, err := opts.FetchManifest(context.Background(), newRepoMock(nil, nil), "mocked-reference")
	if err == nil {
		t.Fatalf("Should fail FetchReference, unexpected return value: %v", ret)
	}

	// Should throw err when repo is empty when parsing invalid platform string
	opts = Platform{"INV@LID_PLATFORM"}
	ret, err = opts.FetchManifest(
		context.Background(),
		newRepoMock([]string{index}, []string{ocispec.MediaTypeImageIndex}),
		digest.FromBytes([]byte(index)).String())
	if err == nil {
		t.Fatalf("Should fail parsing the platform, unexpected return value: %v", ret)
	}

}
