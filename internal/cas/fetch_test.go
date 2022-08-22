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

package cas_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras/internal/cas"
	"oras.land/oras/internal/mock"
)

const (
	index = `{"manifests":[{"digest":"sha256:baf0239e48ff4c47ebac3ba02b5cf1506b69cd5a0c0d0c825a53ba65976fb942","mediaType":"application\/vnd.docker.distribution.manifest.v2+json","platform":{"architecture":"amd64","os":"linux"},"size":11},{"digest":"sha256:27cb13102d774dc36e0bc93f528db7e4f004a6e9636cb6926b1e389668535309","mediaType":"application\/vnd.docker.distribution.manifest.v2+json","platform":{"architecture":"arm","os":"linux","variant":"v5"},"size":12}]}`
	amd64 = "linux/amd64"
	armv5 = "linux/arm/v5"
	armv7 = "linux/arm/v7"

	indexDesc   = `{"mediaType":"application/vnd.oci.image.index.v1+json","digest":"sha256:bdcc003fa2d7882789773fe5fee506ef370dce5ce7988fd420587f144fc700db","size":452}`
	armv5Desc   = `{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","digest":"sha256:27cb13102d774dc36e0bc93f528db7e4f004a6e9636cb6926b1e389668535309","size":12}`
	amd64Desc   = `{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","digest":"sha256:baf0239e48ff4c47ebac3ba02b5cf1506b69cd5a0c0d0c825a53ba65976fb942","size":11}`
	badType     = "application/a.not.supported.manifest.v2+jso"
	badDesc     = `{"mediaType":"application/a.not.supported.manifest.v2+json","digest":"sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","size":0}`
	blobContent = `Hello World`
)

var repo = mock.New().WithFetch().WithFetchReference().WithResolve()

func TestPlatform_FetchManifest_indexAndPlatform(t *testing.T) {
	repo.Remount([]mock.Blob{
		{Content: index, MediaType: ocispec.MediaTypeImageIndex, Tag: ""},
		{Content: amd64, MediaType: ocispec.MediaTypeImageManifest, Tag: ""},
		{Content: armv5, MediaType: ocispec.MediaTypeImageManifest, Tag: ""}})

	// Get index manifest
	indexBytes := []byte(index)
	got, err := cas.FetchManifest(context.Background(), repo, digest.FromBytes(indexBytes).String(), nil)
	if err != nil || !bytes.Equal(got, indexBytes) {
		t.Fatal(err)
	}

	// Get manifest for specific platform
	want := []byte(amd64)
	got, err = cas.FetchManifest(context.Background(), repo, digest.FromBytes(indexBytes).String(), &ocispec.Platform{OS: "linux", Architecture: "amd64"})
	if err != nil || !bytes.Equal(got, want) {
		t.Fatal(err)
	}

	want = []byte(armv5)
	got, err = cas.FetchManifest(context.Background(), repo, digest.FromBytes(indexBytes).String(), &ocispec.Platform{OS: "linux", Architecture: "arm", Variant: "v5"})
	if err != nil || !bytes.Equal(got, want) {
		t.Fatal(err)
	}
}

func TestPlatform_FetchDescriptor_indexAndPlatform(t *testing.T) {
	var indexTag = "multi-platform"
	repo.Remount([]mock.Blob{
		{Content: index, MediaType: ocispec.MediaTypeImageIndex, Tag: indexTag},
		{Content: amd64, MediaType: ocispec.MediaTypeImageManifest, Tag: ""},
		{Content: armv5, MediaType: ocispec.MediaTypeImageManifest, Tag: ""}})

	// Get index manifest
	indexBytes := []byte(index)
	got, err := cas.FetchDescriptor(context.Background(), repo, digest.FromBytes(indexBytes).String(), nil)
	if err != nil || !bytes.Equal(got, []byte(indexDesc)) {
		t.Fatal(err)
	}

	// Get manifest for specific platform
	want := []byte(amd64Desc)
	got, err = cas.FetchDescriptor(context.Background(), repo, indexTag, &ocispec.Platform{OS: "linux", Architecture: "amd64"})
	if err != nil || !bytes.Equal(got, want) {
		t.Fatal(err)
	}
	got, err = cas.FetchDescriptor(context.Background(), repo, indexTag, &ocispec.Platform{OS: "linux", Architecture: "arm", Variant: "v5"})
	if err != nil || !bytes.Equal(got, []byte(armv5Desc)) {
		t.Fatal(err)
	}
}

func TestPlatform_FetchManifest_errNotMulti(t *testing.T) {
	repo.Remount([]mock.Blob{{Content: "", MediaType: badType, Tag: badDesc}})

	// Unknow media type
	_, err := cas.FetchManifest(context.Background(), repo, digest.FromBytes([]byte("")).String(), &ocispec.Platform{OS: "linux", Architecture: "amd64"})
	if !errors.Is(err, errdef.ErrUnsupported) {
		t.Fatalf("Expecting error: %v, got: %v", errdef.ErrUnsupported, err)
	}
}
func TestPlatform_FetchManifest_errNoMatch(t *testing.T) {
	// No matched platform found
	repo.Remount([]mock.Blob{{Content: index, MediaType: ocispec.MediaTypeImageIndex, Tag: ""}})
	_, err := cas.FetchManifest(
		context.Background(),
		repo,
		digest.FromBytes([]byte(index)).String(),
		&ocispec.Platform{OS: "linux", Architecture: "arm", Variant: "v7"})
	if !errors.Is(err, errdef.ErrNotFound) {
		t.Fatalf("Expecting error: %v, got: %v", errdef.ErrNotFound, err)
	}
}

func TestPlatform_FetchDescriptor_miscErr(t *testing.T) {
	// Should throw err when repo is nil
	repo.Remount(nil)
	ret, err := cas.FetchDescriptor(context.Background(), repo, "invalid-RefERENCE", nil)
	if err == nil {
		t.Fatalf("Should fail oras.Resolve, unexpected return value: %v", ret)
	}

}

func TestPlatform_FetchManifest_miscErr(t *testing.T) {
	// Should throw err when repo is empty
	repo.Remount(nil)
	ret, err := cas.FetchManifest(context.Background(), repo, "mocked-reference", nil)
	if err == nil {
		t.Fatalf("Should fail oras.Resolve, unexpected return value: %v", ret)
	}
	// Should throw err when resolve succeeds but fetch reference fails
	tmpRepo := mock.New().WithResolve()
	tmpRepo.Remount([]mock.Blob{{Content: amd64, MediaType: ocispec.MediaTypeImageManifest, Tag: ""}})
	ret, err = cas.FetchManifest(context.Background(), tmpRepo, digest.FromBytes([]byte(amd64)).String(), nil)
	if err == nil {
		t.Fatalf("Should fail oras.Fetch, unexpected return value: %v", ret)
	}
}

func Test_FetchBlob(t *testing.T) {
	repo.Remount([]mock.Blob{
		{Content: blobContent, MediaType: "application/octet-stream", Tag: ""}})

	// Get blob
	contentBytes := []byte(blobContent)
	got, err := cas.FetchBlob(context.Background(), repo, digest.FromBytes(contentBytes).String())
	if err != nil || !bytes.Equal(got, contentBytes) {
		t.Fatal(err)
	}
}
