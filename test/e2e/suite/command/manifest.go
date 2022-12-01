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

package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	. "oras.land/oras/test/e2e/internal/utils"
)

const (
	preview_desc                    = "** This command is in preview and under development. **"
	example_desc                    = "\nExample - "
	repo                            = "command/images"
	multiImage                      = "multi"
	digest_multi                    = "sha256:e2bfc9cc6a84ec2d7365b5a28c6bc5806b7fa581c9ad7883be955a64e3cc034f"
	manifest_multi                  = `{"mediaType":"application/vnd.oci.image.index.v1+json","schemaVersion":2,"manifests":[{"mediaType":"application/vnd.oci.image.manifest.v1+json","digest":"sha256:9d84a5716c66a1d1b9c13f8ed157ba7d1edfe7f9b8766728b8a1f25c0d9c14c1","size":458,"platform":{"architecture":"amd64","os":"linux"}},{"mediaType":"application/vnd.oci.image.manifest.v1+json","digest":"sha256:4f93460061882467e6fb3b772dc6ab72130d9ac1906aed2fc7589a5cd145433c","size":458,"platform":{"architecture":"arm64","os":"linux"}},{"mediaType":"application/vnd.oci.image.manifest.v1+json","digest":"sha256:58efe73e78fe043ca31b89007a025c594ce12aa7e6da27d21c7b14b50112e255","size":458,"platform":{"architecture":"arm","os":"linux","variant":"v7"}}]}`
	descriptor_multi                = `{"mediaType":"application/vnd.oci.image.index.v1+json","digest":"sha256:e2bfc9cc6a84ec2d7365b5a28c6bc5806b7fa581c9ad7883be955a64e3cc034f","size":706}`
	descriptor_linuxAMD64           = `{"mediaType":"application/vnd.oci.image.manifest.v1+json","digest":"sha256:9d84a5716c66a1d1b9c13f8ed157ba7d1edfe7f9b8766728b8a1f25c0d9c14c1","size":458}`
	descriptor_linuxAMD64_fromIndex = `{"mediaType":"application/vnd.oci.image.manifest.v1+json","digest":"sha256:9d84a5716c66a1d1b9c13f8ed157ba7d1edfe7f9b8766728b8a1f25c0d9c14c1","size":458,"platform":{"architecture":"amd64","os":"linux"}}`
	digest_linuxAMD64               = "sha256:9d84a5716c66a1d1b9c13f8ed157ba7d1edfe7f9b8766728b8a1f25c0d9c14c1"
	manifest_linuxAMD64             = `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"mediaType":"application/vnd.oci.image.config.v1+json","digest":"sha256:fe9dbc99451d0517d65e048c309f0b5afb2cc513b7a3d456b6cc29fe641386c5","size":53},"layers":[{"mediaType":"application/vnd.oci.image.layer.v1.tar","digest":"sha256:2ef548696ac7dd66ef38aab5cc8fc5cc1fb637dfaedb3a9afc89bf16db9277e1","size":10240,"annotations":{"org.opencontainers.image.title":"hello.tar"}}]}`
)

var _ = Describe("ORAS beginners:", func() {
	When("running manifest command", func() {
		runAndShowPreviewInHelp := func(args []string, keywords ...string) {
			It(fmt.Sprintf("should run %q command", strings.Join(args, " ")), func() {
				ORAS(append(args, "--help")...).
					MatchKeyWords(append(keywords, "[Preview] "+args[len(args)-1], "\nUsage:")...).
					WithDescription("show preview and help doc").
					Exec()
			})
		}
		runAndShowPreviewInHelp([]string{"manifest"})

		When("running `manifest push`", func() {
			runAndShowPreviewInHelp([]string{"manifest", "push"}, preview_desc, example_desc)
			It("should have flag for prettifying JSON output", func() {
				ORAS("manifest", "push", "--help").
					MatchKeyWords("--pretty", "prettify JSON").
					Exec()
			})

			It("should fail pushing without reference provided", func() {
				ORAS("manifest", "push").
					WithFailureCheck().
					MatchErrKeyWords("Error:").
					Exec()
			})
		})

		When("running `manifest fetch`", func() {
			runAndShowPreviewInHelp([]string{"manifest", "fetch"}, preview_desc, example_desc)
			It("should call sub-commands with aliases", func() {
				ORAS("manifest", "get", "--help").
					MatchKeyWords("[Preview] Fetch", preview_desc, example_desc).
					Exec()
			})
			It("should fail fetching manifest without reference provided", func() {
				ORAS("manifest", "fetch").
					WithFailureCheck().
					MatchErrKeyWords("Error:").
					Exec()
			})
		})
	})
})

var _ = Describe("Common registry users:", func() {
	When("running `manifest fetch`", func() {
		It("should fetch manifest list with digest", func() {
			ORAS("manifest", "fetch", Reference(Host, repo, multiImage)).
				MatchContent(manifest_multi).Exec()
		})

		It("should fetch manifest list with tag", func() {
			ORAS("manifest", "fetch", Reference(Host, repo, multiImage)).
				MatchContent(manifest_multi).Exec()
		})

		It("should fetch manifest list to stdout", func() {
			ORAS("manifest", "fetch", Reference(Host, repo, multiImage), "--output", "-").
				MatchContent(manifest_multi).Exec()
		})

		It("should fetch manifest to file and output descriptor to stdout", func() {
			fetchPath := filepath.Join(GinkgoT().TempDir(), "fetchedImage")
			ORAS("manifest", "fetch", Reference(Host, repo, multiImage), "--output", fetchPath, "--descriptor").
				MatchContent(descriptor_multi).Exec()
			MatchFile(fetchPath, manifest_multi, DefaultTimeout)
		})

		It("should fetch manifest via tag with platform selection", func() {
			ORAS("manifest", "fetch", Reference(Host, repo, multiImage), "--platform", "linux/amd64").
				MatchContent(manifest_linuxAMD64).Exec()
		})

		It("should fetch manifest via digest with platform selection", func() {
			ORAS("manifest", "fetch", Reference(Host, repo, digest_multi), "--platform", "linux/amd64").
				MatchContent(manifest_linuxAMD64).Exec()
		})

		It("should fetch manifest with platform validation", func() {
			ORAS("manifest", "fetch", Reference(Host, repo, digest_linuxAMD64), "--platform", "linux/amd64").
				MatchContent(manifest_linuxAMD64).Exec()
		})

		It("should fetch descriptor via digest", func() {
			ORAS("manifest", "fetch", Reference(Host, repo, digest_multi), "--descriptor").
				MatchContent(descriptor_multi).Exec()
		})

		It("should fetch descriptor via digest with platform selection", func() {
			ORAS("manifest", "fetch", Reference(Host, repo, digest_multi), "--platform", "linux/amd64", "--descriptor").
				MatchContent(descriptor_linuxAMD64_fromIndex).Exec()
		})

		It("should fetch descriptor via digest with platform validation", func() {
			ORAS("manifest", "fetch", Reference(Host, repo, digest_linuxAMD64), "--platform", "linux/amd64", "--descriptor").
				MatchContent(descriptor_linuxAMD64).Exec()
		})

		It("should fetch descriptor via tag", func() {
			ORAS("manifest", "fetch", Reference(Host, repo, digest_multi), "--descriptor").
				MatchContent(descriptor_multi).Exec()
		})

		It("should fetch descriptor via tag with platform selection", func() {
			ORAS("manifest", "fetch", Reference(Host, repo, digest_multi), "--platform", "linux/amd64", "--descriptor").
				MatchContent(descriptor_linuxAMD64_fromIndex).Exec()
		})

		It("should fetch index content with media type assertion", func() {
			ORAS("manifest", "fetch", Reference(Host, repo, digest_multi), "--media-type", "application/vnd.oci.image.index.v1+json").
				MatchContent(manifest_multi).Exec()
		})

		It("should fetch index descriptor with media type assertion", func() {
			ORAS("manifest", "fetch", Reference(Host, repo, digest_multi), "--media-type", "application/vnd.oci.image.index.v1+json", "--descriptor").
				MatchContent(descriptor_multi).Exec()
		})

		It("should fetch image content with media type assertion and platform selection", func() {
			ORAS("manifest", "fetch", Reference(Host, repo, multiImage), "--platform", "linux/amd64", "--media-type", "application/vnd.oci.image.index.v1+json,application/vnd.oci.image.manifest.v1+json").
				MatchContent(manifest_linuxAMD64).Exec()
			ORAS("manifest", "fetch", Reference(Host, repo, digest_multi), "--platform", "linux/amd64", "--media-type", "application/vnd.oci.image.index.v1+json,application/vnd.oci.image.manifest.v1+json", "--descriptor").
				MatchContent(descriptor_linuxAMD64_fromIndex).Exec()
		})

		It("should fetch image descriptor with media type assertion and platform selection", func() {
			ORAS("manifest", "fetch", Reference(Host, repo, multiImage), "--platform", "linux/amd64", "--media-type", "application/vnd.oci.image.index.v1+json,application/vnd.oci.image.manifest.v1+json", "--descriptor").
				MatchContent(descriptor_linuxAMD64_fromIndex).Exec()
			ORAS("manifest", "fetch", Reference(Host, repo, digest_multi), "--platform", "linux/amd64", "--media-type", "application/vnd.oci.image.index.v1+json,application/vnd.oci.image.manifest.v1+json", "--descriptor").
				MatchContent(descriptor_linuxAMD64_fromIndex).Exec()
		})

		It("should fetch image content with media type assertion and platform validation", func() {
			ORAS("manifest", "fetch", Reference(Host, repo, digest_linuxAMD64), "--platform", "linux/amd64", "--media-type", "application/vnd.oci.image.manifest.v1+json").
				MatchContent(manifest_linuxAMD64).Exec()
		})

		It("should fetch image descriptor with media type assertion and platform validation", func() {
			ORAS("manifest", "fetch", Reference(Host, repo, digest_linuxAMD64), "--platform", "linux/amd64", "--media-type", "application/vnd.oci.image.manifest.v1+json", "--descriptor").
				MatchContent(descriptor_linuxAMD64).Exec()
		})

		It("should fail to fetch image if media type assertion fails", func() {
			ORAS("manifest", "fetch", Reference(Host, repo, digest_linuxAMD64), "--media-type", "this.will.not.be.found").
				WithFailureCheck().
				MatchErrKeyWords(digest_linuxAMD64, "error: ", "not found").Exec()
		})
	})

	When("running `manifest push`", func() {
		manifest := `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"mediaType":"application/vnd.oci.image.config.v1+json","digest":"sha256:fe9dbc99451d0517d65e048c309f0b5afb2cc513b7a3d456b6cc29fe641386c5","size":53},"layers":[]}`
		digest := "sha256:bc1a59d49fc7c7b0a31f22ca0c743ecdabdb736777e3d9672fa9d97b4fe323f4"
		descriptor := "{\"mediaType\":\"application/vnd.oci.image.manifest.v1+json\",\"digest\":\"sha256:bc1a59d49fc7c7b0a31f22ca0c743ecdabdb736777e3d9672fa9d97b4fe323f4\",\"size\":247}"

		It("should push a manifest from stdin without media type flag", func() {
			tag := "from-stdin"
			ORAS("manifest", "push", Reference(Host, repo, tag), "-").
				MatchKeyWords("Pushed", Reference(Host, repo, tag), "Digest:", digest).
				WithInput(strings.NewReader(manifest)).Exec()
		})

		It("should push a manifest and output descriptor", func() {
			tag := "from-stdin"
			ORAS("manifest", "push", Reference(Host, repo, tag), "-", "--descriptor").
				MatchContent(descriptor).
				WithInput(strings.NewReader(manifest)).Exec()
		})

		It("should push a manifest from file", func() {
			tempDir := GinkgoT().TempDir()
			manifestPath := filepath.Join(tempDir, "manifest.json")
			err := os.WriteFile(manifestPath, []byte(manifest), 0777)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			tag := "from-file"
			ORAS("manifest", "push", Reference(Host, repo, tag), manifestPath, "--media-type", ocispec.MediaTypeImageManifest).
				MatchKeyWords("Pushed", Reference(Host, repo, tag), "Digest:", digest).
				WithInput(strings.NewReader(manifest)).Exec()
		})

		It("should push a manifest from stdin with media type flag", func() {
			manifest := `{"schemaVersion":2,"config":{"mediaType":"application/vnd.oci.image.config.v1+json","digest":"sha256:fe9dbc99451d0517d65e048c309f0b5afb2cc513b7a3d456b6cc29fe641386c5","size":53}}`
			digest := "sha256:0c2ae2c73c5dde0a42582d328b2e2ea43f36ba20f604fa8706f441ac8b0a3445"
			tag := "mediatype-flag"
			ORAS("manifest", "push", Reference(Host, repo, tag), "-", "--media-type", ocispec.MediaTypeImageManifest).
				MatchKeyWords("Pushed", Reference(Host, repo, tag), "Digest:", digest).
				WithInput(strings.NewReader(manifest)).Exec()

			ORAS("manifest", "push", Reference(Host, repo, ""), "-").
				WithInput(strings.NewReader(manifest)).
				WithFailureCheck().
				WithDescription("fail if no media type flag provided").Exec()
		})
	})
})
