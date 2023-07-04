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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/test/e2e/internal/testdata/feature"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	"oras.land/oras/test/e2e/internal/testdata/multi_arch"
	. "oras.land/oras/test/e2e/internal/utils"
)

func prepare(src string, dst string) {
	ORAS("cp", src, dst).WithDescription("prepare test env").Exec()
}

func validateTag(repoRef string, tag string, gone bool) {
	session := ORAS("repo", "tags", repoRef).Exec()
	quoted := regexp.QuoteMeta(tag + "\n")
	if gone {
		Expect(session.Out).NotTo(gbytes.Say(quoted))
	} else {
		Expect(session.Out).To(gbytes.Say(quoted))
	}
}

var _ = Describe("ORAS beginners:", func() {
	repoFmt := fmt.Sprintf("command/manifest/%%s/%d/%%s", GinkgoRandomSeed())
	When("running manifest command", func() {
		When("running `manifest push`", func() {
			It("should show help doc with feature flags", func() {
				out := ORAS("manifest", "push", "--help").MatchKeyWords(ExampleDesc).Exec()
				gomega.Expect(out).Should(gbytes.Say("--distribution-spec string\\s+%s", regexp.QuoteMeta(feature.Preview.Mark)))
			})

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

			It("should fail pushing with  a manifest from stdin without media type flag", func() {
				tag := "from-stdin"
				ORAS("manifest", "push", RegistryRef(Host, ImageRepo, tag), "-", "--password-stdin", "--media-type", "application/vnd.oci.image.manifest.v1+json").
					ExpectFailure().
					MatchErrKeyWords("`-`", "`--password-stdin`", " cannot be both used").Exec()
			})
		})

		When("running `manifest fetch`", func() {
			It("should call sub-commands with aliases", func() {
				ORAS("manifest", "get", "--help").
					MatchKeyWords(ExampleDesc).
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
			It("should show help doc with feature flags", func() {
				out := ORAS("manifest", "delete", "--help").MatchKeyWords(ExampleDesc).Exec()
				gomega.Expect(out).Should(gbytes.Say("--distribution-spec string\\s+%s", regexp.QuoteMeta(feature.Preview.Mark)))
			})

			tempTag := "to-delete"
			It("should cancel deletion without confirmation", func() {
				dstRepo := fmt.Sprintf(repoFmt, "delete", "no-confirm")
				prepare(RegistryRef(Host, ImageRepo, foobar.Tag), RegistryRef(Host, dstRepo, tempTag))
				ORAS("manifest", "delete", RegistryRef(Host, dstRepo, tempTag)).
					MatchKeyWords("Operation cancelled.", "Are you sure you want to delete the manifest ", " and all tags associated with it?").Exec()
				validateTag(RegistryRef(Host, dstRepo, ""), tempTag, false)
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

			It("should fail if no manifest reference provided", func() {
				dstRepo := fmt.Sprintf(repoFmt, "delete", "no-reference")
				prepare(RegistryRef(Host, ImageRepo, foobar.Tag), RegistryRef(Host, dstRepo, tempTag))
				ORAS("manifest", "delete").ExpectFailure().Exec()
			})

			It("should fail if no digest provided", func() {
				dstRepo := fmt.Sprintf(repoFmt, "delete", "no-reference")
				prepare(RegistryRef(Host, ImageRepo, foobar.Tag), RegistryRef(Host, dstRepo, ""))
				ORAS("manifest", "delete", RegistryRef(Host, dstRepo, "")).ExpectFailure().MatchErrKeyWords("name@digest").Exec()
			})
		})
		When("running `manifest fetch-config`", func() {
			It("should show preview hint in the doc", func() {
				ORAS("manifest", "fetch-config", "--help").
					MatchKeyWords(ExampleDesc, "\nUsage:").Exec()
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
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.Digest)).
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
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.Tag), "--descriptor").
				MatchContent(multi_arch.Descriptor).Exec()
		})

		It("should fetch descriptor via tag with platform selection", func() {
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.Tag), "--platform", "linux/amd64", "--descriptor").
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
		It("should fail if no manifest tag or digest is provided", func() {
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, "")).ExpectFailure().MatchErrKeyWords("Error:", "invalid image reference").Exec()
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
			ORAS("manifest", "push", RegistryRef(Host, ImageRepo, tag), manifestPath, "--media-type", "application/vnd.oci.image.manifest.v1+json").
				MatchKeyWords("Pushed", RegistryRef(Host, ImageRepo, tag), "Digest:", digest).
				WithInput(strings.NewReader(manifest)).Exec()
		})

		It("should push a manifest from stdin with media type flag", func() {
			manifest := `{"schemaVersion":2,"config":{"mediaType":"application/vnd.oci.image.config.v1+json","digest":"sha256:fe9dbc99451d0517d65e048c309f0b5afb2cc513b7a3d456b6cc29fe641386c5","size":53}}`
			digest := "sha256:0c2ae2c73c5dde0a42582d328b2e2ea43f36ba20f604fa8706f441ac8b0a3445"
			tag := "mediatype-flag"
			ORAS("manifest", "push", RegistryRef(Host, ImageRepo, tag), "-", "--media-type", "application/vnd.oci.image.manifest.v1+json").
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
				MatchContent(foobar.ImageConfigDesc).Exec()
		})

		It("should fetch a config via digest", func() {
			ORAS("manifest", "fetch-config", RegistryRef(Host, ImageRepo, foobar.Tag)).
				MatchContent("{}").Exec()
		})

		It("should fetch a config descriptor via a digest", func() {
			ORAS("manifest", "fetch-config", "--descriptor", RegistryRef(Host, ImageRepo, foobar.Digest)).
				MatchContent(foobar.ImageConfigDesc).Exec()
		})

		It("should fetch a config of a specific platform", func() {
			ORAS("manifest", "fetch-config", "--platform", "linux/amd64", RegistryRef(Host, ImageRepo, multi_arch.Tag)).
				MatchContent(multi_arch.LinuxAMD64Config).Exec()
		})

		It("should fetch a config descriptor of a specific platform", func() {
			ORAS("manifest", "fetch-config", "--descriptor", "--platform", "linux/amd64", RegistryRef(Host, ImageRepo, multi_arch.Tag)).
				MatchContent(multi_arch.LinuxAMD64ConfigDesc).Exec()
		})
		It("should fail if no manifest tag or digest is provided", func() {
			ORAS("manifest", "fetch-config", RegistryRef(Host, ImageRepo, "")).ExpectFailure().MatchErrKeyWords("Error:", "invalid image reference").Exec()
		})
	})

	When("running `manifest delete`", func() {
		tempTag := "to-delete"
		It("should do confirmed deletion via input", func() {
			dstRepo := fmt.Sprintf(repoFmt, "delete", "confirm-input")
			prepare(RegistryRef(Host, ImageRepo, foobar.Tag), RegistryRef(Host, dstRepo, tempTag))
			ORAS("manifest", "delete", RegistryRef(Host, dstRepo, tempTag)).
				WithInput(strings.NewReader("y")).Exec()
			validateTag(RegistryRef(Host, dstRepo, ""), tempTag, true)
		})

		It("should do confirmed deletion via flag", func() {
			dstRepo := fmt.Sprintf(repoFmt, "delete", "confirm-flag")
			prepare(RegistryRef(Host, ImageRepo, foobar.Tag), RegistryRef(Host, dstRepo, tempTag))
			ORAS("manifest", "delete", RegistryRef(Host, dstRepo, tempTag), "-f").Exec()
			validateTag(RegistryRef(Host, dstRepo, ""), tempTag, true)
		})

		It("should do forced deletion and output descriptor", func() {
			dstRepo := fmt.Sprintf(repoFmt, "delete", "output-descriptor")
			prepare(RegistryRef(Host, ImageRepo, foobar.Tag), RegistryRef(Host, dstRepo, tempTag))
			ORAS("manifest", "delete", RegistryRef(Host, dstRepo, tempTag), "-f", "--descriptor").
				MatchContent("{\"mediaType\":\"application/vnd.oci.image.manifest.v1+json\",\"digest\":\"sha256:fd6ed2f36b5465244d5dc86cb4e7df0ab8a9d24adc57825099f522fe009a22bb\",\"size\":851}").
				WithDescription("cancel without confirmation").Exec()
			validateTag(RegistryRef(Host, dstRepo, ""), tempTag, true)
		})

		It("should succeed when deleting a non-existent manifest with force flag set", func() {
			toDeleteRef := RegistryRef(Host, ImageRepo, invalidDigest)
			ORAS("manifest", "delete", toDeleteRef, "--force").
				MatchKeyWords("Missing", toDeleteRef).
				Exec()
		})
	})
})

var _ = Describe("OCI image layout users:", func() {
	When("running `manifest fetch`", func() {
		prepare := func() string {
			tmp := GinkgoT().TempDir()
			ORAS("cp", RegistryRef(Host, ImageRepo, multi_arch.Digest), Flags.ToLayout, LayoutRef(tmp, multi_arch.Tag)).WithDescription("prepare image from registry to OCI layout").Exec()
			return tmp
		}
		It("should fetch manifest list with digest", func() {
			root := prepare()
			ORAS("manifest", "fetch", Flags.Layout, LayoutRef(root, multi_arch.Digest)).
				MatchContent(multi_arch.Manifest).Exec()
		})
		It("should fetch manifest list with tag", func() {
			root := prepare()
			ORAS("manifest", "fetch", Flags.Layout, LayoutRef(root, multi_arch.Tag)).
				MatchContent(multi_arch.Manifest).Exec()
		})
		It("should fetch manifest list to stdout", func() {
			root := prepare()
			ORAS("manifest", "fetch", Flags.Layout, LayoutRef(root, multi_arch.Tag), "--output", "-").
				MatchContent(multi_arch.Manifest).Exec()
		})
		It("should fetch manifest to file and output descriptor to stdout", func() {
			root := prepare()
			fetchPath := filepath.Join(GinkgoT().TempDir(), "fetchedImage")
			ORAS("manifest", "fetch", Flags.Layout, LayoutRef(root, multi_arch.Digest), "--output", fetchPath, "--descriptor").
				MatchContent(multi_arch.Descriptor).Exec()
			MatchFile(fetchPath, multi_arch.Manifest, DefaultTimeout)
		})
		It("should fetch manifest via tag with platform selection", func() {
			root := prepare()
			ORAS("manifest", "fetch", Flags.Layout, LayoutRef(root, multi_arch.Tag), "--platform", "linux/amd64").
				MatchContent(multi_arch.LinuxAMD64Manifest).Exec()
		})
		It("should fetch manifest via digest with platform selection", func() {
			root := prepare()
			ORAS("manifest", "fetch", Flags.Layout, LayoutRef(root, multi_arch.Digest), "--platform", "linux/amd64").
				MatchContent(multi_arch.LinuxAMD64Manifest).Exec()
		})
		It("should fetch manifest with platform validation", func() {
			root := prepare()
			ORAS("manifest", "fetch", Flags.Layout, LayoutRef(root, multi_arch.Digest), "--platform", "linux/amd64").
				MatchContent(multi_arch.LinuxAMD64Manifest).Exec()
		})
		It("should fetch descriptor via digest", func() {
			root := prepare()
			ORAS("manifest", "fetch", Flags.Layout, LayoutRef(root, multi_arch.Digest), "--descriptor").
				MatchContent(multi_arch.Descriptor).Exec()
		})
		It("should fetch descriptor via digest with platform selection", func() {
			root := prepare()
			ORAS("manifest", "fetch", Flags.Layout, LayoutRef(root, multi_arch.Digest),
				"--platform", "linux/amd64", "--descriptor").
				MatchContent(multi_arch.LinuxAMD64IndexDesc).Exec()
		})
		It("should fetch descriptor via digest with platform validation", func() {
			root := prepare()
			ORAS("manifest", "fetch", Flags.Layout, LayoutRef(root, multi_arch.LinuxAMD64.Digest.String()),
				"--platform", "linux/amd64", "--descriptor").
				MatchContent(multi_arch.LinuxAMD64DescStr).Exec()
		})
		It("should fetch descriptor via tag", func() {
			root := prepare()
			ORAS("manifest", "fetch", Flags.Layout, LayoutRef(root, multi_arch.Tag), "--descriptor").
				MatchContent(multi_arch.AnnotatedDescriptor).Exec()
		})
		It("should fetch descriptor via tag with platform selection", func() {
			root := prepare()
			ORAS("manifest", "fetch", Flags.Layout, LayoutRef(root, multi_arch.Tag),
				"--platform", "linux/amd64", "--descriptor").
				MatchContent(multi_arch.LinuxAMD64IndexDesc).Exec()
		})
		It("should fail to fetch image if media type assertion is used", func() {
			root := prepare()
			ORAS("manifest", "fetch", Flags.Layout, LayoutRef(root, multi_arch.Digest), "--media-type", "application/vnd.oci.image.manifest.v1+json").
				ExpectFailure().
				MatchErrKeyWords("Error", "--media-type", "--oci-layout").Exec()
		})
		It("should fail if no manifest tag or digest is provided", func() {
			root := prepare()
			ORAS("manifest", "fetch", Flags.Layout, root).ExpectFailure().
				MatchErrKeyWords("Error:", "invalid image reference").Exec()
		})
	})

	When("running `manifest fetch-config`", func() {
		prepare := func(tag string) string {
			tmpRoot := GinkgoT().TempDir()
			cpPath := tmpRoot
			from := RegistryRef(Host, ImageRepo, tag)
			cpPath = fmt.Sprintf("%s:%s", tmpRoot, tag)
			ORAS("cp", from, Flags.ToLayout, cpPath).WithDescription("prepare image from registry to OCI layout").Exec()
			return tmpRoot
		}
		It("should fetch a config via a tag", func() {
			root := prepare(foobar.Tag)
			ORAS("manifest", "fetch-config", Flags.Layout, LayoutRef(root, foobar.Tag)).
				MatchContent("{}").Exec()
		})
		It("should fetch a config descriptor via a tag", func() {
			root := prepare(foobar.Tag)
			ORAS("manifest", "fetch-config", "--descriptor", Flags.Layout, LayoutRef(root, foobar.Tag)).
				MatchContent(foobar.ImageConfigDesc).Exec()
		})
		It("should fetch a config via digest", func() {
			root := prepare(foobar.Tag)
			ORAS("manifest", "fetch-config", Flags.Layout, LayoutRef(root, foobar.Digest)).
				MatchContent("{}").Exec()
		})
		It("should fetch a config descriptor via a digest", func() {
			root := prepare(foobar.Tag)
			ORAS("manifest", "fetch-config", "--descriptor", Flags.Layout, LayoutRef(root, foobar.Digest)).
				MatchContent(foobar.ImageConfigDesc).Exec()
		})
		It("should fetch a config of a specific platform", func() {
			root := prepare(multi_arch.Tag)
			ORAS("manifest", "fetch-config", "--platform", "linux/amd64", Flags.Layout, LayoutRef(root, multi_arch.Tag)).
				MatchContent(multi_arch.LinuxAMD64Config).Exec()
		})
		It("should fetch a config descriptor of a specific platform", func() {
			root := prepare(multi_arch.Tag)
			ORAS("manifest", "fetch-config", "--descriptor", "--platform", "linux/amd64", Flags.Layout, LayoutRef(root, multi_arch.Tag)).
				MatchContent(multi_arch.LinuxAMD64ConfigDesc).Exec()
		})
		It("should fail if no manifest tag or digest is provided", func() {
			root := prepare(foobar.Tag)
			ORAS("manifest", "fetch-config", Flags.Layout, root).ExpectFailure().MatchErrKeyWords("Error:", "invalid image reference").Exec()
		})
	})

	When("running `manifest push`", func() {
		scratchSize := 2
		scratchDigest := "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a"
		manifest := fmt.Sprintf(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"mediaType":"application/vnd.oci.image.config.v1+json","digest":"%s","size":%d},"layers":[]}`, scratchDigest, scratchSize)
		manifestDigest := "sha256:f20c43161d73848408ef247f0ec7111b19fe58ffebc0cbcaa0d2c8bda4967268"
		prepare := func(layoutRoot string) {
			ORAS("blob", "push", Flags.Layout, LayoutRef(layoutRoot, scratchDigest), "--size", "2", "-").
				WithInput(strings.NewReader("{}")).Exec()
		}
		validate := func(root string, digest string, tag string) {
			path := filepath.Join(root, "index.json")
			Expect(path).To(BeAnExistingFile())
			content, err := os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
			var index ocispec.Index
			Expect(json.Unmarshal(content, &index)).ShouldNot(HaveOccurred())
			for _, m := range index.Manifests {
				if m.Digest.String() == digest &&
					(tag == "" || tag == m.Annotations["org.opencontainers.image.ref.name"]) {
					return
				}
			}
			Fail(fmt.Sprintf("Failed to find manifest with digest %q and tag %q in index.json: \n%s", digest, tag, string(content)))
		}
		descriptor := "{\"mediaType\":\"application/vnd.oci.image.manifest.v1+json\",\"digest\":\"sha256:f20c43161d73848408ef247f0ec7111b19fe58ffebc0cbcaa0d2c8bda4967268\",\"size\":246}"

		It("should push and output verbosed warning for Feferrers deletion by default", func() {
			manifestPath := WriteTempFile("manifest.json", manifest)
			root := filepath.Dir(manifestPath)
			prepare(root)
			ORAS("manifest", "push", root, Flags.Layout, manifestPath, "--skip-delete-referrers=false").
				WithWorkDir(root).
				MatchErrKeyWords("referrers deletion can only be enforced upon registry\n").Exec()
		})

		It("should push a manifest from stdin", func() {
			root := GinkgoT().TempDir()
			prepare(root)
			ORAS("manifest", "push", Flags.Layout, root, "-").
				MatchKeyWords("Pushed", root, "Digest:", manifestDigest).
				WithInput(strings.NewReader(manifest)).Exec()
			validate(root, manifestDigest, "")
		})

		It("should push a manifest from stdin and tag", func() {
			tag := "from-stdin"
			root := GinkgoT().TempDir()
			ref := LayoutRef(root, tag)
			ORAS("manifest", "push", Flags.Layout, ref, "-").
				MatchKeyWords("Pushed", ref, "Digest:", manifestDigest).
				WithInput(strings.NewReader(manifest)).Exec()
			validate(root, manifestDigest, tag)
		})

		It("should push a manifest and output descriptor", func() {
			root := GinkgoT().TempDir()
			prepare(root)
			ORAS("manifest", "push", Flags.Layout, root, "-", "--descriptor").
				MatchContent(descriptor).
				WithInput(strings.NewReader(manifest)).Exec()
			validate(root, manifestDigest, "")
		})

		It("should push a manifest from file", func() {
			manifestPath := WriteTempFile("manifest.json", manifest)
			root := filepath.Dir(manifestPath)
			prepare(root)
			tag := "from-file"
			ref := LayoutRef(root, tag)
			ORAS("manifest", "push", Flags.Layout, ref, manifestPath).
				MatchKeyWords("Pushed", ref, "Digest:", manifestDigest).
				WithInput(strings.NewReader(manifest)).Exec()
			validate(root, manifestDigest, tag)
		})

		It("should push a manifest from stdin, only when media type flag is set", func() {
			manifest := fmt.Sprintf(`{"schemaVersion":2,"config":{"mediaType":"application/vnd.oci.image.config.v1+json","digest":"%s","size":%d}}`, scratchDigest, scratchSize)
			manifestDigest := "sha256:8fc649142bbc0a2aa5015d5ef5a922df9d2d7f2dcf3095dbebfaf7c271eca444"

			root := GinkgoT().TempDir()
			prepare(root)
			tag := "mediatype-flag"
			ref := LayoutRef(root, tag)
			ORAS("manifest", "push", Flags.Layout, ref, "-", "--media-type", "application/vnd.oci.image.manifest.v1+json").
				MatchKeyWords("Pushed", ref, "Digest:", manifestDigest).
				WithInput(strings.NewReader(manifest)).Exec()
			validate(root, manifestDigest, tag)

			ORAS("manifest", "push", Flags.Layout, ref, "-").
				WithInput(strings.NewReader(manifest)).
				ExpectFailure().
				WithDescription("fail if no media type flag provided").Exec()
		})
	})
})

func pushTestRepo(text string) string {
	return fmt.Sprintf("command/push/%d/%s", GinkgoRandomSeed(), text)
}

var _ = Describe("Fallback registry users:", func() {
	When("running `manifest push`", func() {
		manifest := fmt.Sprintf(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"mediaType":"oras.test","digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2},"subject":%s,"layers":[]}`, foobar.DescriptorStr)
		It("should fail to push manifest when cleaning referrers index", func() {
			testRepo := pushTestRepo("fallback/fail-gc")
			subjectRef := RegistryRef(FallbackHost, testRepo, foobar.Tag)
			// prepare
			ORAS("cp", RegistryRef(FallbackHost, ArtifactRepo, foobar.Tag), subjectRef, "-r").Exec()
			// test
			ORAS("manifest", "push", RegistryRef(FallbackHost, testRepo, ""), "-").
				WithInput(strings.NewReader(manifest)).
				ExpectFailure().
				MatchErrKeyWords("failed to delete dangling referrers index").
				Exec()
		})
		It("should push manifest and skip cleaning referrers index", func() {
			testRepo := pushTestRepo("fallback/skip-gc")
			subjectRef := RegistryRef(FallbackHost, testRepo, foobar.Tag)
			// prepare
			ORAS("cp", RegistryRef(FallbackHost, ArtifactRepo, foobar.Tag), subjectRef, "-r").Exec()
			// test
			ORAS("manifest", "push", RegistryRef(FallbackHost, testRepo, ""), "-", "--skip-delete-referrers").
				WithInput(strings.NewReader(manifest)).
				Exec()
			// validate
			var index ocispec.Index
			bytes := ORAS("discover", subjectRef, "-o", "json").Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(2))
		})
	})
})
