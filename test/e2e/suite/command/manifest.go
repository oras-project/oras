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
	. "oras.land/oras/test/e2e/internal/utils"
)

func prepare(src string, dst string) {
	ORAS("cp", src, dst).Exec()
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
				prepare(Reference(Host, Repo, FoobarImageTag), Reference(Host, dstRepo, tempTag))
				ORAS("manifest", "delete", Reference(Host, dstRepo, tempTag)).
					MatchKeyWords("Operation cancelled.", "Are you sure you want to delete the manifest ", " and all tags associated with it?").Exec()
				validate(Reference(Host, dstRepo, ""), tempTag, false)
			})

			It("should fail if descriptor flag is provided without confirmation flag", func() {
				dstRepo := fmt.Sprintf(repoFmt, "delete", "descriptor-without-confirm")
				prepare(Reference(Host, Repo, FoobarImageTag), Reference(Host, dstRepo, tempTag))
				ORAS("manifest", "delete", Reference(Host, dstRepo, tempTag), "--descriptor").ExpectFailure().Exec()
			})

			It("should fail to delete a non-existent manifest via digest without force flag set", func() {
				toDeleteRef := Reference(Host, Repo, invalidDigest)
				ORAS("manifest", "delete", toDeleteRef).
					ExpectFailure().
					MatchErrKeyWords(toDeleteRef, "the specified manifest does not exist").
					Exec()
			})

			It("should fail to delete a non-existent manifest via tag without force flag set", func() {
				toDeleteRef := Reference(Host, Repo, "this.tag.should-not.be-existed")
				ORAS("manifest", "delete", toDeleteRef).
					ExpectFailure().
					MatchErrKeyWords(toDeleteRef, "the specified manifest does not exist").
					Exec()
			})

			It("should fail if no blob reference provided", func() {
				dstRepo := fmt.Sprintf(repoFmt, "delete", "no-reference")
				prepare(Reference(Host, Repo, FoobarImageTag), Reference(Host, dstRepo, tempTag))
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
				ORAS("manifest", "fetch-config", Reference(Host, Repo, "this-tag-should-not-exist")).ExpectFailure().Exec()
			})
			It("should fail fetching a config of non-image manifest type", func() {
				ORAS("manifest", "fetch-config", Reference(Host, Repo, MultiImageTag)).ExpectFailure().Exec()
			})
		})
	})
})

var _ = Describe("Common registry users:", func() {
	repoFmt := fmt.Sprintf("command/manifest/%%s/%d/%%s", GinkgoRandomSeed())
	When("running `manifest fetch`", func() {
		It("should fetch manifest list with digest", func() {
			ORAS("manifest", "fetch", Reference(Host, Repo, MultiImageTag)).
				MatchContent(MultiImageManifest).Exec()
		})

		It("should fetch manifest list with tag", func() {
			ORAS("manifest", "fetch", Reference(Host, Repo, MultiImageTag)).
				MatchContent(MultiImageManifest).Exec()
		})

		It("should fetch manifest list to stdout", func() {
			ORAS("manifest", "fetch", Reference(Host, Repo, MultiImageTag), "--output", "-").
				MatchContent(MultiImageManifest).Exec()
		})

		It("should fetch manifest to file and output descriptor to stdout", func() {
			fetchPath := filepath.Join(GinkgoT().TempDir(), "fetchedImage")
			ORAS("manifest", "fetch", Reference(Host, Repo, MultiImageTag), "--output", fetchPath, "--descriptor").
				MatchContent(MultiImageDescriptor).Exec()
			MatchFile(fetchPath, MultiImageManifest, DefaultTimeout)
		})

		It("should fetch manifest via tag with platform selection", func() {
			ORAS("manifest", "fetch", Reference(Host, Repo, MultiImageTag), "--platform", "linux/amd64").
				MatchContent(LinuxAMD64ImageManifest).Exec()
		})

		It("should fetch manifest via digest with platform selection", func() {
			ORAS("manifest", "fetch", Reference(Host, Repo, MultiImageDigest), "--platform", "linux/amd64").
				MatchContent(LinuxAMD64ImageManifest).Exec()
		})

		It("should fetch manifest with platform validation", func() {
			ORAS("manifest", "fetch", Reference(Host, Repo, LinuxAMD64ImageDigest), "--platform", "linux/amd64").
				MatchContent(LinuxAMD64ImageManifest).Exec()
		})

		It("should fetch descriptor via digest", func() {
			ORAS("manifest", "fetch", Reference(Host, Repo, MultiImageDigest), "--descriptor").
				MatchContent(MultiImageDescriptor).Exec()
		})

		It("should fetch descriptor via digest with platform selection", func() {
			ORAS("manifest", "fetch", Reference(Host, Repo, MultiImageDigest), "--platform", "linux/amd64", "--descriptor").
				MatchContent(LinuxAMD64ImageIndexDescriptor).Exec()
		})

		It("should fetch descriptor via digest with platform validation", func() {
			ORAS("manifest", "fetch", Reference(Host, Repo, LinuxAMD64ImageDigest), "--platform", "linux/amd64", "--descriptor").
				MatchContent(LinuxAMD64ImageDescriptor).Exec()
		})

		It("should fetch descriptor via tag", func() {
			ORAS("manifest", "fetch", Reference(Host, Repo, MultiImageDigest), "--descriptor").
				MatchContent(MultiImageDescriptor).Exec()
		})

		It("should fetch descriptor via tag with platform selection", func() {
			ORAS("manifest", "fetch", Reference(Host, Repo, MultiImageDigest), "--platform", "linux/amd64", "--descriptor").
				MatchContent(LinuxAMD64ImageIndexDescriptor).Exec()
		})

		It("should fetch index content with media type assertion", func() {
			ORAS("manifest", "fetch", Reference(Host, Repo, MultiImageDigest), "--media-type", "application/vnd.oci.image.index.v1+json").
				MatchContent(MultiImageManifest).Exec()
		})

		It("should fetch index descriptor with media type assertion", func() {
			ORAS("manifest", "fetch", Reference(Host, Repo, MultiImageDigest), "--media-type", "application/vnd.oci.image.index.v1+json", "--descriptor").
				MatchContent(MultiImageDescriptor).Exec()
		})

		It("should fetch image content with media type assertion and platform selection", func() {
			ORAS("manifest", "fetch", Reference(Host, Repo, MultiImageTag), "--platform", "linux/amd64", "--media-type", "application/vnd.oci.image.index.v1+json,application/vnd.oci.image.manifest.v1+json").
				MatchContent(LinuxAMD64ImageManifest).Exec()
			ORAS("manifest", "fetch", Reference(Host, Repo, MultiImageDigest), "--platform", "linux/amd64", "--media-type", "application/vnd.oci.image.index.v1+json,application/vnd.oci.image.manifest.v1+json", "--descriptor").
				MatchContent(LinuxAMD64ImageIndexDescriptor).Exec()
		})

		It("should fetch image descriptor with media type assertion and platform selection", func() {
			ORAS("manifest", "fetch", Reference(Host, Repo, MultiImageTag), "--platform", "linux/amd64", "--media-type", "application/vnd.oci.image.index.v1+json,application/vnd.oci.image.manifest.v1+json", "--descriptor").
				MatchContent(LinuxAMD64ImageIndexDescriptor).Exec()
			ORAS("manifest", "fetch", Reference(Host, Repo, MultiImageDigest), "--platform", "linux/amd64", "--media-type", "application/vnd.oci.image.index.v1+json,application/vnd.oci.image.manifest.v1+json", "--descriptor").
				MatchContent(LinuxAMD64ImageIndexDescriptor).Exec()
		})

		It("should fetch image content with media type assertion and platform validation", func() {
			ORAS("manifest", "fetch", Reference(Host, Repo, LinuxAMD64ImageDigest), "--platform", "linux/amd64", "--media-type", "application/vnd.oci.image.manifest.v1+json").
				MatchContent(LinuxAMD64ImageManifest).Exec()
		})

		It("should fetch image descriptor with media type assertion and platform validation", func() {
			ORAS("manifest", "fetch", Reference(Host, Repo, LinuxAMD64ImageDigest), "--platform", "linux/amd64", "--media-type", "application/vnd.oci.image.manifest.v1+json", "--descriptor").
				MatchContent(LinuxAMD64ImageDescriptor).Exec()
		})

		It("should fail to fetch image if media type assertion fails", func() {
			ORAS("manifest", "fetch", Reference(Host, Repo, LinuxAMD64ImageDigest), "--media-type", "this.will.not.be.found").
				ExpectFailure().
				MatchErrKeyWords(LinuxAMD64ImageDigest, "error: ", "not found").Exec()
		})
	})

	When("running `manifest push`", func() {
		manifest := `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"mediaType":"application/vnd.oci.image.config.v1+json","digest":"sha256:fe9dbc99451d0517d65e048c309f0b5afb2cc513b7a3d456b6cc29fe641386c5","size":53},"layers":[]}`
		digest := "sha256:bc1a59d49fc7c7b0a31f22ca0c743ecdabdb736777e3d9672fa9d97b4fe323f4"
		descriptor := "{\"mediaType\":\"application/vnd.oci.image.manifest.v1+json\",\"digest\":\"sha256:bc1a59d49fc7c7b0a31f22ca0c743ecdabdb736777e3d9672fa9d97b4fe323f4\",\"size\":247}"

		It("should push a manifest from stdin without media type flag", func() {
			tag := "from-stdin"
			ORAS("manifest", "push", Reference(Host, Repo, tag), "-").
				MatchKeyWords("Pushed", Reference(Host, Repo, tag), "Digest:", digest).
				WithInput(strings.NewReader(manifest)).Exec()
		})

		It("should push a manifest and output descriptor", func() {
			tag := "from-stdin"
			ORAS("manifest", "push", Reference(Host, Repo, tag), "-", "--descriptor").
				MatchContent(descriptor).
				WithInput(strings.NewReader(manifest)).Exec()
		})

		It("should push a manifest from file", func() {
			manifestPath := WriteTempFile("manifest.json", manifest)
			tag := "from-file"
			ORAS("manifest", "push", Reference(Host, Repo, tag), manifestPath, "--media-type", ocispec.MediaTypeImageManifest).
				MatchKeyWords("Pushed", Reference(Host, Repo, tag), "Digest:", digest).
				WithInput(strings.NewReader(manifest)).Exec()
		})

		It("should push a manifest from stdin with media type flag", func() {
			manifest := `{"schemaVersion":2,"config":{"mediaType":"application/vnd.oci.image.config.v1+json","digest":"sha256:fe9dbc99451d0517d65e048c309f0b5afb2cc513b7a3d456b6cc29fe641386c5","size":53}}`
			digest := "sha256:0c2ae2c73c5dde0a42582d328b2e2ea43f36ba20f604fa8706f441ac8b0a3445"
			tag := "mediatype-flag"
			ORAS("manifest", "push", Reference(Host, Repo, tag), "-", "--media-type", ocispec.MediaTypeImageManifest).
				MatchKeyWords("Pushed", Reference(Host, Repo, tag), "Digest:", digest).
				WithInput(strings.NewReader(manifest)).Exec()

			ORAS("manifest", "push", Reference(Host, Repo, ""), "-").
				WithInput(strings.NewReader(manifest)).
				ExpectFailure().
				WithDescription("fail if no media type flag provided").Exec()
		})
	})

	When("running `manifest fetch-config`", func() {
		It("should fetch a config via a tag", func() {
			ORAS("manifest", "fetch-config", Reference(Host, Repo, FoobarImageTag)).
				MatchContent("{}").Exec()
		})

		It("should fetch a config descriptor via a tag", func() {
			ORAS("manifest", "fetch-config", "--descriptor", Reference(Host, Repo, FoobarImageTag)).
				MatchContent(FoobarConfigDesc).Exec()
		})

		It("should fetch a config via digest", func() {
			ORAS("manifest", "fetch-config", Reference(Host, Repo, FoobarImageTag)).
				MatchContent("{}").Exec()
		})

		It("should fetch a config descriptor via a digest", func() {
			ORAS("manifest", "fetch-config", "--descriptor", Reference(Host, Repo, FoobarImageDigest)).
				MatchContent(FoobarConfigDesc).Exec()
		})

		It("should fetch a config of a specific platform", func() {
			ORAS("manifest", "fetch-config", "--platform", "linux/amd64", Reference(Host, Repo, MultiImageTag)).
				MatchContent(LinuxAMD64ImageConfig).Exec()
		})

		It("should fetch a config descriptor of a specific platform", func() {
			ORAS("manifest", "fetch-config", "--descriptor", "--platform", "linux/amd64", Reference(Host, Repo, MultiImageTag)).
				MatchContent(LinuxAMD64ImageConfigDescriptor).Exec()
		})
	})

	When("running `manifest delete`", func() {
		tempTag := "to-delete"
		It("should do confirmed deletion via input", func() {
			dstRepo := fmt.Sprintf(repoFmt, "delete", "confirm-input")
			prepare(Reference(Host, Repo, FoobarImageTag), Reference(Host, dstRepo, tempTag))
			ORAS("manifest", "delete", Reference(Host, dstRepo, tempTag)).
				WithInput(strings.NewReader("y")).Exec()
			validate(Reference(Host, dstRepo, ""), tempTag, true)
		})

		It("should do confirmed deletion via flag", func() {
			dstRepo := fmt.Sprintf(repoFmt, "delete", "confirm-flag")
			prepare(Reference(Host, Repo, FoobarImageTag), Reference(Host, dstRepo, tempTag))
			ORAS("manifest", "delete", Reference(Host, dstRepo, tempTag), "-f").Exec()
			validate(Reference(Host, dstRepo, ""), tempTag, true)
		})

		It("should do confirmed deletion and output descriptor", func() {
			dstRepo := fmt.Sprintf(repoFmt, "delete", "output-descriptor")
			prepare(Reference(Host, Repo, FoobarImageTag), Reference(Host, dstRepo, tempTag))
			ORAS("manifest", "delete", Reference(Host, dstRepo, tempTag), "-f", "--descriptor").
				MatchContent("{\"mediaType\":\"application/vnd.oci.image.manifest.v1+json\",\"digest\":\"sha256:fd6ed2f36b5465244d5dc86cb4e7df0ab8a9d24adc57825099f522fe009a22bb\",\"size\":851}").
				WithDescription("cancel without confirmation").Exec()
			validate(Reference(Host, dstRepo, ""), tempTag, true)
		})

		It("should succeed when deleting a non-existent manifest with force flag set", func() {
			toDeleteRef := Reference(Host, Repo, invalidDigest)
			ORAS("manifest", "delete", toDeleteRef, "--force").
				MatchErrKeyWords(toDeleteRef, "the specified manifest does not exist").
				Exec()
		})
	})
})
