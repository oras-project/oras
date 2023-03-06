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
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	"oras.land/oras/test/e2e/internal/testdata/multi_arch"
	. "oras.land/oras/test/e2e/internal/utils"
)

func prepare(src string, dst string) {
	ORAS("cp", src, dst).WithDescription("prepare test env").Exec()
}

func validate(repoRef string, tag string, gone bool) {
	session := ORAS("repo", "tags", repoRef).Exec()
	if gone {
		Expect(session.Out).NotTo(gbytes.Say(tag))
	} else {
		Expect(session.Out).To(gbytes.Say(tag))
	}
}

var _ = Describe("ORAS beginners:", func() {
	repoFmt := fmt.Sprintf("command/manifest/%%s/%d/%%s", GinkgoRandomSeed())
	When("running manifest command", func() {
		RunAndShowPreviewInHelp([]string{"manifest"})

		When("running `manifest push`", func() {
			RunAndShowPreviewInHelp([]string{"manifest", "push"}, PreviewDesc, ExampleDesc)
			It("should have flag for prettifying JSON output", func() {
				ORAS("manifest", "push", "--help").
					MatchKeyWords("--pretty", "prettify JSON").
					Exec()
			})

			It("should fail pushing without reference provided", func() {
				ORAS("manifest", "push").
					ExpectFailure().
					MatchErrKeyWords("Error:").
					Exec()
			})
		})

		When("running `manifest fetch`", func() {
			RunAndShowPreviewInHelp([]string{"manifest", "fetch"}, PreviewDesc, ExampleDesc)
			It("should call sub-commands with aliases", func() {
				ORAS("manifest", "get", "--help").
					MatchKeyWords("[Preview] Fetch", PreviewDesc, ExampleDesc).
					Exec()
			})
			It("should fail fetching manifest without reference provided", func() {
				ORAS("manifest", "fetch").
					ExpectFailure().
					MatchErrKeyWords("Error:").
					Exec()
			})
		})
		When("running `manifest delete`", func() {
			tempTag := "to-delete"
			It("should cancel deletion without confirmation", func() {
				dstRepo := fmt.Sprintf(repoFmt, "delete", "no-confirm")
				prepare(RegistryRef(Host, ImageRepo, foobar.Tag), RegistryRef(Host, dstRepo, tempTag))
				ORAS("manifest", "delete", RegistryRef(Host, dstRepo, tempTag)).
					MatchKeyWords("Operation cancelled.", "Are you sure you want to delete the manifest ", " and all tags associated with it?").Exec()
				validate(RegistryRef(Host, dstRepo, ""), tempTag, false)
			})

			It("should fail if descriptor flag is provided without confirmation flag", func() {
				dstRepo := fmt.Sprintf(repoFmt, "delete", "descriptor-without-confirm")
				prepare(RegistryRef(Host, ImageRepo, foobar.Tag), RegistryRef(Host, dstRepo, tempTag))
				ORAS("manifest", "delete", RegistryRef(Host, dstRepo, tempTag), "--descriptor").ExpectFailure().Exec()
			})

			It("should fail to delete a non-existent manifest via digest without force flag set", func() {
				toDeleteRef := RegistryRef(Host, ImageRepo, invalidDigest)
				ORAS("manifest", "delete", toDeleteRef).
					ExpectFailure().
					MatchErrKeyWords(toDeleteRef, "the specified manifest does not exist").
					Exec()
			})

			It("should fail to delete a non-existent manifest and output descriptor via digest, with force flag set", func() {
				toDeleteRef := RegistryRef(Host, ImageRepo, invalidDigest)
				ORAS("manifest", "delete", toDeleteRef, "--force", "--descriptor").
					ExpectFailure().
					MatchErrKeyWords(toDeleteRef, "the specified manifest does not exist").
					Exec()
			})

			It("should fail to delete a non-existent manifest and output descriptor via tag, without force flag set", func() {
				toDeleteRef := RegistryRef(Host, ImageRepo, "this.tag.should-not.be-existed")
				ORAS("manifest", "delete", toDeleteRef, "--force", "--descriptor").
					ExpectFailure().
					MatchErrKeyWords(toDeleteRef, "the specified manifest does not exist").
					Exec()
			})

			It("should fail if no blob reference provided", func() {
				dstRepo := fmt.Sprintf(repoFmt, "delete", "no-reference")
				prepare(RegistryRef(Host, ImageRepo, foobar.Tag), RegistryRef(Host, dstRepo, tempTag))
				ORAS("manifest", "delete").ExpectFailure().Exec()
			})
		})
		When("running `manifest fetch-config`", func() {
			It("should show preview hint in the doc", func() {
				ORAS("manifest", "fetch-config", "--help").
					MatchKeyWords(PreviewDesc, ExampleDesc, "[Preview]", "\nUsage:").Exec()
			})

			It("should fail if no manifest reference provided", func() {
				ORAS("manifest", "fetch-config").ExpectFailure().Exec()
			})

			It("should fail if provided reference does not exist", func() {
				ORAS("manifest", "fetch-config", RegistryRef(Host, ImageRepo, "this-tag-should-not-exist")).ExpectFailure().Exec()
			})
			It("should fail fetching a config of non-image manifest type", func() {
				ORAS("manifest", "fetch-config", RegistryRef(Host, ImageRepo, multi_arch.Tag)).ExpectFailure().Exec()
			})
		})
	})
})

var _ = Describe("Common registry users:", func() {
	repoFmt := fmt.Sprintf("command/manifest/%%s/%d/%%s", GinkgoRandomSeed())
	When("running `manifest fetch`", func() {
		It("should fetch manifest list with digest", func() {
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.Tag)).
				MatchContent(multi_arch.Manifest).Exec()
		})

		It("should fetch manifest list with tag", func() {
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.Tag)).
				MatchContent(multi_arch.Manifest).Exec()
		})

		It("should fetch manifest list to stdout", func() {
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.Tag), "--output", "-").
				MatchContent(multi_arch.Manifest).Exec()
		})

		It("should fetch manifest to file and output descriptor to stdout", func() {
			fetchPath := filepath.Join(GinkgoT().TempDir(), "fetchedImage")
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.Tag), "--output", fetchPath, "--descriptor").
				MatchContent(multi_arch.Descriptor).Exec()
			MatchFile(fetchPath, multi_arch.Manifest, DefaultTimeout)
		})

		It("should fetch manifest via tag with platform selection", func() {
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.Tag), "--platform", "linux/amd64").
				MatchContent(multi_arch.LinuxAMD64Manifest).Exec()
		})

		It("should fetch manifest via digest with platform selection", func() {
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.Digest), "--platform", "linux/amd64").
				MatchContent(multi_arch.LinuxAMD64Manifest).Exec()
		})

		It("should fetch manifest with platform validation", func() {
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.LinuxAMD64.Digest.String()), "--platform", "linux/amd64").
				MatchContent(multi_arch.LinuxAMD64Manifest).Exec()
		})

		It("should fetch descriptor via digest", func() {
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.Digest), "--descriptor").
				MatchContent(multi_arch.Descriptor).Exec()
		})

		It("should fetch descriptor via digest with platform selection", func() {
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.Digest), "--platform", "linux/amd64", "--descriptor").
				MatchContent(multi_arch.LinuxAMD64IndexDesc).Exec()
		})

		It("should fetch descriptor via digest with platform validation", func() {
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.LinuxAMD64.Digest.String()), "--platform", "linux/amd64", "--descriptor").
				MatchContent(multi_arch.LinuxAMD64DescStr).Exec()
		})

		It("should fetch descriptor via tag", func() {
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.Digest), "--descriptor").
				MatchContent(multi_arch.Descriptor).Exec()
		})

		It("should fetch descriptor via tag with platform selection", func() {
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.Digest), "--platform", "linux/amd64", "--descriptor").
				MatchContent(multi_arch.LinuxAMD64IndexDesc).Exec()
		})

		It("should fetch index content with media type assertion", func() {
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.Digest), "--media-type", "application/vnd.oci.image.index.v1+json").
				MatchContent(multi_arch.Manifest).Exec()
		})

		It("should fetch index descriptor with media type assertion", func() {
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.Digest), "--media-type", "application/vnd.oci.image.index.v1+json", "--descriptor").
				MatchContent(multi_arch.Descriptor).Exec()
		})

		It("should fetch image content with media type assertion and platform selection", func() {
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.Tag), "--platform", "linux/amd64", "--media-type", "application/vnd.oci.image.index.v1+json,application/vnd.oci.image.manifest.v1+json").
				MatchContent(multi_arch.LinuxAMD64Manifest).Exec()
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.Digest), "--platform", "linux/amd64", "--media-type", "application/vnd.oci.image.index.v1+json,application/vnd.oci.image.manifest.v1+json", "--descriptor").
				MatchContent(multi_arch.LinuxAMD64IndexDesc).Exec()
		})

		It("should fetch image descriptor with media type assertion and platform selection", func() {
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.Tag), "--platform", "linux/amd64", "--media-type", "application/vnd.oci.image.index.v1+json,application/vnd.oci.image.manifest.v1+json", "--descriptor").
				MatchContent(multi_arch.LinuxAMD64IndexDesc).Exec()
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.Digest), "--platform", "linux/amd64", "--media-type", "application/vnd.oci.image.index.v1+json,application/vnd.oci.image.manifest.v1+json", "--descriptor").
				MatchContent(multi_arch.LinuxAMD64IndexDesc).Exec()
		})

		It("should fetch image content with media type assertion and platform validation", func() {
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.LinuxAMD64.Digest.String()), "--platform", "linux/amd64", "--media-type", "application/vnd.oci.image.manifest.v1+json").
				MatchContent(multi_arch.LinuxAMD64Manifest).Exec()
		})

		It("should fetch image descriptor with media type assertion and platform validation", func() {
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.LinuxAMD64.Digest.String()), "--platform", "linux/amd64", "--media-type", "application/vnd.oci.image.manifest.v1+json", "--descriptor").
				MatchContent(multi_arch.LinuxAMD64DescStr).Exec()
		})

		It("should fail to fetch image if media type assertion fails", func() {
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.LinuxAMD64.Digest.String()), "--media-type", "this.will.not.be.found").
				ExpectFailure().
				MatchErrKeyWords(multi_arch.LinuxAMD64.Digest.String(), "error: ", "not found").Exec()
		})
	})

	When("running `manifest push`", func() {
		manifest := `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"mediaType":"application/vnd.oci.image.config.v1+json","digest":"sha256:fe9dbc99451d0517d65e048c309f0b5afb2cc513b7a3d456b6cc29fe641386c5","size":53},"layers":[]}`
		digest := "sha256:bc1a59d49fc7c7b0a31f22ca0c743ecdabdb736777e3d9672fa9d97b4fe323f4"
		descriptor := "{\"mediaType\":\"application/vnd.oci.image.manifest.v1+json\",\"digest\":\"sha256:bc1a59d49fc7c7b0a31f22ca0c743ecdabdb736777e3d9672fa9d97b4fe323f4\",\"size\":247}"

		It("should push a manifest from stdin without media type flag", func() {
			tag := "from-stdin"
			ORAS("manifest", "push", RegistryRef(Host, ImageRepo, tag), "-").
				MatchKeyWords("Pushed", RegistryRef(Host, ImageRepo, tag), "Digest:", digest).
				WithInput(strings.NewReader(manifest)).Exec()
		})

		It("should push a manifest and output descriptor", func() {
			tag := "from-stdin"
			ORAS("manifest", "push", RegistryRef(Host, ImageRepo, tag), "-", "--descriptor").
				MatchContent(descriptor).
				WithInput(strings.NewReader(manifest)).Exec()
		})

		It("should push a manifest from file", func() {
			manifestPath := WriteTempFile("manifest.json", manifest)
			tag := "from-file"
			ORAS("manifest", "push", RegistryRef(Host, ImageRepo, tag), manifestPath, "--media-type", ocispec.MediaTypeImageManifest).
				MatchKeyWords("Pushed", RegistryRef(Host, ImageRepo, tag), "Digest:", digest).
				WithInput(strings.NewReader(manifest)).Exec()
		})

		It("should push a manifest from stdin with media type flag", func() {
			manifest := `{"schemaVersion":2,"config":{"mediaType":"application/vnd.oci.image.config.v1+json","digest":"sha256:fe9dbc99451d0517d65e048c309f0b5afb2cc513b7a3d456b6cc29fe641386c5","size":53}}`
			digest := "sha256:0c2ae2c73c5dde0a42582d328b2e2ea43f36ba20f604fa8706f441ac8b0a3445"
			tag := "mediatype-flag"
			ORAS("manifest", "push", RegistryRef(Host, ImageRepo, tag), "-", "--media-type", ocispec.MediaTypeImageManifest).
				MatchKeyWords("Pushed", RegistryRef(Host, ImageRepo, tag), "Digest:", digest).
				WithInput(strings.NewReader(manifest)).Exec()

			ORAS("manifest", "push", RegistryRef(Host, ImageRepo, ""), "-").
				WithInput(strings.NewReader(manifest)).
				ExpectFailure().
				WithDescription("fail if no media type flag provided").Exec()
		})
	})

	When("running `manifest fetch-config`", func() {
		It("should fetch a config via a tag", func() {
			ORAS("manifest", "fetch-config", RegistryRef(Host, ImageRepo, foobar.Tag)).
				MatchContent("{}").Exec()
		})

		It("should fetch a config descriptor via a tag", func() {
			ORAS("manifest", "fetch-config", "--descriptor", RegistryRef(Host, ImageRepo, foobar.Tag)).
				MatchContent(foobar.ConfigDesc).Exec()
		})

		It("should fetch a config via digest", func() {
			ORAS("manifest", "fetch-config", RegistryRef(Host, ImageRepo, foobar.Tag)).
				MatchContent("{}").Exec()
		})

		It("should fetch a config descriptor via a digest", func() {
			ORAS("manifest", "fetch-config", "--descriptor", RegistryRef(Host, ImageRepo, foobar.Digest)).
				MatchContent(foobar.ConfigDesc).Exec()
		})

		It("should fetch a config of a specific platform", func() {
			ORAS("manifest", "fetch-config", "--platform", "linux/amd64", RegistryRef(Host, ImageRepo, multi_arch.Tag)).
				MatchContent(multi_arch.LinuxAMD64Config).Exec()
		})

		It("should fetch a config descriptor of a specific platform", func() {
			ORAS("manifest", "fetch-config", "--descriptor", "--platform", "linux/amd64", RegistryRef(Host, ImageRepo, multi_arch.Tag)).
				MatchContent(multi_arch.LinuxAMD64ConfigDesc).Exec()
		})
	})

	When("running `manifest delete`", func() {
		tempTag := "to-delete"
		It("should do confirmed deletion via input", func() {
			dstRepo := fmt.Sprintf(repoFmt, "delete", "confirm-input")
			prepare(RegistryRef(Host, ImageRepo, foobar.Tag), RegistryRef(Host, dstRepo, tempTag))
			ORAS("manifest", "delete", RegistryRef(Host, dstRepo, tempTag)).
				WithInput(strings.NewReader("y")).Exec()
			validate(RegistryRef(Host, dstRepo, ""), tempTag, true)
		})

		It("should do confirmed deletion via flag", func() {
			dstRepo := fmt.Sprintf(repoFmt, "delete", "confirm-flag")
			prepare(RegistryRef(Host, ImageRepo, foobar.Tag), RegistryRef(Host, dstRepo, tempTag))
			ORAS("manifest", "delete", RegistryRef(Host, dstRepo, tempTag), "-f").Exec()
			validate(RegistryRef(Host, dstRepo, ""), tempTag, true)
		})

		It("should do confirmed deletion and output descriptor", func() {
			dstRepo := fmt.Sprintf(repoFmt, "delete", "output-descriptor")
			prepare(RegistryRef(Host, ImageRepo, foobar.Tag), RegistryRef(Host, dstRepo, tempTag))
			ORAS("manifest", "delete", RegistryRef(Host, dstRepo, tempTag), "-f", "--descriptor").
				MatchContent("{\"mediaType\":\"application/vnd.oci.image.manifest.v1+json\",\"digest\":\"sha256:fd6ed2f36b5465244d5dc86cb4e7df0ab8a9d24adc57825099f522fe009a22bb\",\"size\":851}").
				WithDescription("cancel without confirmation").Exec()
			validate(RegistryRef(Host, dstRepo, ""), tempTag, true)
		})

		It("should succeed when deleting a non-existent manifest with force flag set", func() {
			toDeleteRef := RegistryRef(Host, ImageRepo, invalidDigest)
			ORAS("manifest", "delete", toDeleteRef, "--force").
				MatchKeyWords("Missing", toDeleteRef).
				Exec()
		})
	})
})
