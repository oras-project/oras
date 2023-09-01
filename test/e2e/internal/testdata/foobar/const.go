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

package foobar

import (
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/test/e2e/internal/utils/match"
)

var (
	Tag              = "foobar"
	Digest           = "sha256:fd6ed2f36b5465244d5dc86cb4e7df0ab8a9d24adc57825099f522fe009a22bb"
	ManifestStateKey = match.StateKey{Digest: "fd6ed2f36b54", Name: "application/vnd.oci.image.manifest.v1+json"}

	FileLayerNames = []string{
		"foobar/foo1",
		"foobar/foo2",
		FileBarName,
	}
	FileConfigName     = "foobar/config.json"
	FileConfigSize     = 13
	FileConfigDigest   = digest.Digest("sha256:46b68ac1696c3870d537f376868d9402400de28587e345264a77b65da09669be")
	FileConfigStateKey = match.StateKey{
		Digest: "46b68ac1696c", Name: "application/vnd.unknown.config.v1+json",
	}

	FileBarName     = "foobar/bar"
	FileBarStateKey = match.StateKey{Digest: "fcde2b2edba5", Name: FileLayerNames[2]}
	FileStateKeys   = []match.StateKey{
		{Digest: "2c26b46b68ff", Name: FileLayerNames[0]},
		{Digest: "2c26b46b68ff", Name: FileLayerNames[1]},
		FileBarStateKey,
	}

	AttachFileName     = "foobar/to-be-attached"
	AttachFileMedia    = "test/oras.e2e"
	AttachFileStateKey = match.StateKey{
		Digest: "d3b29f7d12d9", Name: AttachFileName,
	}

	ImageConfigDesc = "{\"mediaType\":\"application/vnd.unknown.config.v1+json\",\"digest\":\"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a\",\"size\":2}"
	ImageLayerNames = []string{
		"foo1",
		"foo2",
		"bar",
	}
	ImageLayerStateKeys = []match.StateKey{
		{Digest: "2c26b46b68ff", Name: ImageLayerNames[0]},
		{Digest: "2c26b46b68ff", Name: ImageLayerNames[1]},
		{Digest: "fcde2b2edba5", Name: ImageLayerNames[2]},
	}
	ImageConfigName = "config.json"

	FooBlobDigest       = "sha256:2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae"
	FooBlobContent      = "foo"
	FooBlobDescriptor   = `{"mediaType":"application/octet-stream","digest":"sha256:2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae","size":3}`
	BarBlobDigest       = "sha256:fcde2b2edba56bf408601fb721fe9b5c338d10ee429ea04fae5511b68fbf8fb9"
	BlobBarDescTemplate = `{"mediaType":"%s","digest":"sha256:fcde2b2edba56bf408601fb721fe9b5c338d10ee429ea04fae5511b68fbf8fb9","size":3,"annotations":{"org.opencontainers.image.title":"foobar/bar"}}`
)

func BlobBarDescriptor(mediaType string) ocispec.Descriptor {
	return ocispec.Descriptor{
		MediaType: mediaType,
		Digest:    digest.Digest(BarBlobDigest),
		Size:      3,
		Annotations: map[string]string{
			"org.opencontainers.image.title": FileBarName,
		},
	}
}

func ImageConfigStateKey(configName string) match.StateKey {
	return match.StateKey{Digest: "44136fa355b3", Name: configName}
}

// referrers
var (
	SBOMImageReferrer = ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    digest.Digest("sha256:e2c6633a79985906f1ed55c592718c73c41e809fb9818de232a635904a74d48d"),
		Size:      660,
		Annotations: map[string]string{
			"org.opencontainers.image.created": "2023-01-18T08:37:42Z",
		},
		ArtifactType: "test/sbom.file",
	}
	SignatureImageReferrer = ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    digest.Digest("sha256:0cb8c4da7e9ff2e7eefca33141091b9239218e3125a35e17e8bcd05fa3a5e714"),
		Size:      670,
	}
	SBOMArtifactReferrer = ocispec.Descriptor{
		MediaType: "application/vnd.oci.artifact.manifest.v1+json",
		Digest:    digest.Digest("sha256:8d7a27ff2662dae183f762d281f46d626ba7b6e56a72cc9959cdbcd91aad7fbc"),
		Size:      547,
		Annotations: map[string]string{
			"org.opencontainers.artifact.created": "2023-01-16T05:49:46Z",
		},
		ArtifactType: "test.sbom.file",
	}
	SignatureArtifactReferrer = ocispec.Descriptor{
		MediaType: "application/vnd.oci.artifact.manifest.v1+json",
		Digest:    digest.Digest("sha256:2dbea575a3490375f5052fbeb380a2f498866d99eb809b4168e49e224a274a39"),
		Size:      560,
	}
	ArtifactReferrerStateKeys = []match.StateKey{
		{Digest: "8d7a27ff2662", Name: "application/vnd.oci.artifact.manifest.v1+json"},
		{Digest: "2dbea575a349", Name: "application/vnd.oci.artifact.manifest.v1+json"},
	}

	ImageReferrersStateKeys = []match.StateKey{
		{Digest: "0cb8c4da7e9f", Name: "application/vnd.oci.image.manifest.v1+json"},
		{Digest: "e2c6633a7998", Name: "application/vnd.oci.image.manifest.v1+json"},
	}
	ImageReferrerConfigStateKeys = []match.StateKey{
		{Digest: "44136fa355b3", Name: "test/signature.file"},
		{Digest: "44136fa355b3", Name: "test/sbom.file"},
	}
)
