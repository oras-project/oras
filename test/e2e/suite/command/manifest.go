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
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
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
		runAndShowPreviewInHelp([]string{"manifest"})
		runAndShowPreviewInHelp([]string{"manifest", "fetch"}, preview_desc, example_desc)
		runAndShowPreviewInHelp([]string{"manifest", "push"}, preview_desc, example_desc)

		It("should call sub-commands with aliases", func() {
			ORAS("manifest", "get", "--help").
				MatchKeyWords("[Preview] Fetch", preview_desc, example_desc).
				Exec()
		})
		It("should fetch manifest with no artifact reference provided", func() {
			ORAS("manifest", "fetch").
				ExpectFailure().
				MatchErrKeyWords("Error:").
				Exec()
		})
		It("should have flag for prettifying JSON output", func() {
			ORAS("manifest", "push", "--help").
				MatchKeyWords("--pretty", "prettify JSON").
				Exec()
		})
	})
})

func runAndShowPreviewInHelp(args []string, keywords ...string) {
	It(fmt.Sprintf("should run %q command", strings.Join(args, " ")), func() {
		ORAS(append(args, "--help")...).
			MatchKeyWords(append(keywords, "[Preview] "+args[len(args)-1], "\nUsage:")...).
			WithDescription("show preview and help doc").
			Exec()
	})
}

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
				ExpectFailure().
				MatchErrKeyWords(digest_linuxAMD64, "error: ", "not found").Exec()
		})
	})

	prepare := func(src string, dst string) {
		ORAS("cp", src, dst).Exec()
	}
	validate := func(repoRef string, tag string, gone bool) {
		session := ORAS("repo", "tags", repoRef).Exec()
		if gone {
			Expect(session.Out).NotTo(gbytes.Say(tag))
		} else {
			Expect(session.Out).To(gbytes.Say(tag))
		}
	}
	srcTag := "foobar"
	dstTag := "e2e"
	var repoPrefix = fmt.Sprintf("%s/%d", repo, GinkgoRandomSeed())
	When("running `manifest delete`", func() {
		It("should do confirmed deletion via input", func() {
			dstRepo := fmt.Sprintf("%s/%s", repoPrefix, "confirm-input")
			prepare(Reference(Host, repo, srcTag), Reference(Host, dstRepo, dstTag))
			ORAS("manifest", "delete", Reference(Host, dstRepo, dstTag)).
				WithInput(strings.NewReader("y")).Exec()
			validate(Reference(Host, dstRepo, ""), dstTag, true)
		})

		It("should do confirmed deletion via flag", func() {
			dstRepo := fmt.Sprintf("%s/%s", repoPrefix, "confirm-flag")
			prepare(Reference(Host, repo, srcTag), Reference(Host, dstRepo, dstTag))
			ORAS("manifest", "delete", Reference(Host, dstRepo, dstTag), "-f").Exec()
			validate(Reference(Host, dstRepo, ""), dstTag, true)
		})

		It("should do confirmed deletion and output descriptor", func() {
			dstRepo := fmt.Sprintf("%s/%s", repoPrefix, "output-descriptor")
			prepare(Reference(Host, repo, srcTag), Reference(Host, dstRepo, dstTag))
			ORAS("manifest", "delete", Reference(Host, dstRepo, dstTag), "-f", "--descriptor").
				MatchContent("{\"mediaType\":\"application/vnd.oci.image.manifest.v1+json\",\"digest\":\"sha256:fd6ed2f36b5465244d5dc86cb4e7df0ab8a9d24adc57825099f522fe009a22bb\",\"size\":851}").
				WithDescription("cancel without confirmation").Exec()
			validate(Reference(Host, dstRepo, ""), dstTag, true)
		})
	})

	When("running `manifest delete` with wrong input", func() {
		It("should cancel deletion without confirmation", func() {
			dstRepo := fmt.Sprintf("%s/%s", repoPrefix, "no-confirm")
			prepare(Reference(Host, repo, srcTag), Reference(Host, dstRepo, dstTag))
			ORAS("manifest", "delete", Reference(Host, dstRepo, dstTag)).
				MatchKeyWords("Operation cancelled.", "Are you sure you want to delete the manifest ", " and all tags associated with it?").Exec()
			validate(Reference(Host, dstRepo, ""), dstTag, false)
		})

		It("should fail if descriptor flag is provided without confirmation flag", func() {
			dstRepo := fmt.Sprintf("%s/%s", repoPrefix, "descriptor-without-confirm")
			prepare(Reference(Host, repo, srcTag), Reference(Host, dstRepo, dstTag))
			ORAS("manifest", "delete", Reference(Host, dstRepo, dstTag), "--descriptor").ExpectFailure().Exec()
		})

		It("should fail if no blob reference provided", func() {
			dstRepo := fmt.Sprintf("%s/%s", repoPrefix, "no-reference")
			prepare(Reference(Host, repo, srcTag), Reference(Host, dstRepo, dstTag))
			ORAS("manifest", "delete").ExpectFailure().Exec()
		})
	})
})
