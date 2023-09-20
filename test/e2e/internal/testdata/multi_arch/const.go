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

package multi_arch

import (
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/test/e2e/internal/utils/match"
)

var (
	Tag                         = "multi"
	EmptyTag                    = "empty_index"
	Digest                      = "sha256:e2bfc9cc6a84ec2d7365b5a28c6bc5806b7fa581c9ad7883be955a64e3cc034f"
	Manifest                    = `{"mediaType":"application/vnd.oci.image.index.v1+json","schemaVersion":2,"manifests":[{"mediaType":"application/vnd.oci.image.manifest.v1+json","digest":"sha256:9d84a5716c66a1d1b9c13f8ed157ba7d1edfe7f9b8766728b8a1f25c0d9c14c1","size":458,"platform":{"architecture":"amd64","os":"linux"}},{"mediaType":"application/vnd.oci.image.manifest.v1+json","digest":"sha256:4f93460061882467e6fb3b772dc6ab72130d9ac1906aed2fc7589a5cd145433c","size":458,"platform":{"architecture":"arm64","os":"linux"}},{"mediaType":"application/vnd.oci.image.manifest.v1+json","digest":"sha256:58efe73e78fe043ca31b89007a025c594ce12aa7e6da27d21c7b14b50112e255","size":458,"platform":{"architecture":"arm","os":"linux","variant":"v7"}}]}`
	Descriptor                  = `{"mediaType":"application/vnd.oci.image.index.v1+json","digest":"sha256:e2bfc9cc6a84ec2d7365b5a28c6bc5806b7fa581c9ad7883be955a64e3cc034f","size":706}`
	AnnotatedDescriptor         = `{"mediaType":"application/vnd.oci.image.index.v1+json","digest":"sha256:e2bfc9cc6a84ec2d7365b5a28c6bc5806b7fa581c9ad7883be955a64e3cc034f","size":706,"annotations":{"org.opencontainers.image.ref.name":"multi"}}`
	IndexReferrerDigest         = "sha256:d37baf66300b9006b0f4c7102075d56b970fbf910be5c6bca07fdbb000dfa383"
	IndexReferrerStateKey       = match.StateKey{Digest: "d3cf790759b0", Name: "application/vnd.oci.image.manifest.v1+json"}
	IndexReferrerConfigStateKey = match.StateKey{Digest: "44136fa355b3", Name: "referrer/index"}
	IndexStateKeys              = []match.StateKey{
		{Digest: "2ef548696ac7", Name: "hello.tar"},
		{Digest: "fe9dbc99451d", Name: "application/vnd.oci.image.config.v1+json"},
		{Digest: "9d84a5716c66", Name: "application/vnd.oci.image.manifest.v1+json"},
		{Digest: "e2bfc9cc6a84", Name: "application/vnd.oci.image.index.v1+json"},
	}
	// below are newly baked artifacts for ZOT
	IndexZOTReferrerStateKey = match.StateKey{Digest: "d37baf66300b", Name: "application/vnd.oci.image.manifest.v1+json"}
)

// child images
var (
	LinuxAMD64Manifest = `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"mediaType":"application/vnd.oci.image.config.v1+json","digest":"sha256:fe9dbc99451d0517d65e048c309f0b5afb2cc513b7a3d456b6cc29fe641386c5","size":53},"layers":[{"mediaType":"application/vnd.oci.image.layer.v1.tar","digest":"sha256:2ef548696ac7dd66ef38aab5cc8fc5cc1fb637dfaedb3a9afc89bf16db9277e1","size":10240,"annotations":{"org.opencontainers.image.title":"hello.tar"}}]}`
	LinuxAMD64         = ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    digest.Digest("sha256:9d84a5716c66a1d1b9c13f8ed157ba7d1edfe7f9b8766728b8a1f25c0d9c14c1"),
		Size:      458,
	}
	LinuxAMD64DescStr    = `{"mediaType":"application/vnd.oci.image.manifest.v1+json","digest":"sha256:9d84a5716c66a1d1b9c13f8ed157ba7d1edfe7f9b8766728b8a1f25c0d9c14c1","size":458}`
	LinuxAMD64IndexDesc  = `{"mediaType":"application/vnd.oci.image.manifest.v1+json","digest":"sha256:9d84a5716c66a1d1b9c13f8ed157ba7d1edfe7f9b8766728b8a1f25c0d9c14c1","size":458,"platform":{"architecture":"amd64","os":"linux"}}`
	LinuxAMD64Config     = "{\r\n    \"architecture\": \"amd64\",\r\n    \"os\": \"linux\"\r\n}"
	LinuxAMD64ConfigDesc = `{"mediaType":"application/vnd.oci.image.config.v1+json","digest":"sha256:fe9dbc99451d0517d65e048c309f0b5afb2cc513b7a3d456b6cc29fe641386c5","size":53}`
	LinuxAMD64Referrer   = ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    digest.Digest("sha256:c5e00045954a70e3fd28307dd543d4cc158946117943700b8f520f72ddca031f"),
		Annotations: map[string]string{
			"org.opencontainers.image.created": "2023-02-15T07:56:34Z",
			"subject":                          "linux/amd64",
		},
		ArtifactType: "referrer/image",
		Size:         482,
	}
	LayerName                        = "hello.tar"
	LinuxAMD64ReferrerStateKey       = match.StateKey{Digest: "c5e00045954a", Name: "application/vnd.oci.image.manifest.v1+json"}
	LinuxAMD64ReferrerConfigStateKey = match.StateKey{Digest: "44136fa355b3", Name: "referrer/image"}
	LinuxAMD64StateKeys              = []match.StateKey{
		{Digest: "9d84a5716c66", Name: "application/vnd.oci.image.manifest.v1+json"},
		{Digest: "fe9dbc99451d", Name: "application/vnd.oci.image.config.v1+json"},
		{Digest: "2ef548696ac7", Name: "hello.tar"},
	}
)
