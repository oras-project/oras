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
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras/test/e2e/internal/testdata/artifact/blob"
	"oras.land/oras/test/e2e/internal/testdata/artifact/config"
	"oras.land/oras/test/e2e/internal/testdata/artifact/index"
	"oras.land/oras/test/e2e/internal/testdata/feature"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	ma "oras.land/oras/test/e2e/internal/testdata/multi_arch"
	. "oras.land/oras/test/e2e/internal/utils"
)

func cpTestRepo(text string) string {
	return fmt.Sprintf("command/copy/%d/%s", GinkgoRandomSeed(), text)
}

var _ = Describe("ORAS beginners:", func() {
	When("running cp command", func() {
		It("should show help doc with feature flags", func() {
			out := ORAS("cp", "--help").MatchKeyWords("Copy", ExampleDesc).Exec()
			Expect(out).Should(gbytes.Say("--from-distribution-spec string\\s+%s", regexp.QuoteMeta(feature.Preview.Mark)))
			Expect(out).Should(gbytes.Say("-r, --recursive\\s+%s", regexp.QuoteMeta(feature.Preview.Mark)))
			Expect(out).Should(gbytes.Say("--to-distribution-spec string\\s+%s", regexp.QuoteMeta(feature.Preview.Mark)))
		})

		It("should fail when no reference provided", func() {
			ORAS("cp").ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})

		It("should fail when no destination reference provided", func() {
			ORAS("cp", RegistryRef(ZOTHost, ImageRepo, foobar.Tag)).ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})

		It("should fail when no tag or digest is provided for source target", func() {
			ORAS("cp", RegistryRef(ZOTHost, ImageRepo, ""), RegistryRef(ZOTHost, ImageRepo, "dst")).ExpectFailure().MatchErrKeyWords("Error:", "no tag or digest specified", "oras cp").Exec()
		})

		It("should fail when source doesn't exist", func() {
			ORAS("cp", RegistryRef(ZOTHost, ImageRepo, InvalidTag), RegistryRef(ZOTHost, cpTestRepo("nonexistent-source"), "")).ExpectFailure().MatchErrKeyWords(InvalidTag).Exec()
		})

		It("should fail and show detailed error description if no argument provided", func() {
			err := ORAS("cp").ExpectFailure().Exec().Err
			Expect(err).Should(gbytes.Say("Error"))
			Expect(err).Should(gbytes.Say("\nUsage: oras cp"))
			Expect(err).Should(gbytes.Say("\n"))
			Expect(err).Should(gbytes.Say(`Run "oras cp -h"`))
		})

		It("should fail and show detailed error description if more than 2 arguments are provided", func() {
			err := ORAS("cp", "foo", "bar", "buz").ExpectFailure().Exec().Err
			Expect(err).Should(gbytes.Say("Error"))
			Expect(err).Should(gbytes.Say("\nUsage: oras cp"))
			Expect(err).Should(gbytes.Say("\n"))
			Expect(err).Should(gbytes.Say(`Run "oras cp -h"`))
		})

		It("should fail and show registry error prefix if source not found", func() {
			src := RegistryRef(ZOTHost, ArtifactRepo, InvalidTag)
			dst := GinkgoT().TempDir()
			ORAS("cp", src, Flags.ToLayout, dst).MatchErrKeyWords(RegistryErrorPrefix).ExpectFailure().Exec()
		})

		It("should fail and show registry error prefix if destination registry is not logged in", func() {
			src := PrepareTempOCI(ArtifactRepo)
			dst := RegistryRef(ZOTHost, cpTestRepo("dest-not-logged-in"), "")
			ORAS("cp", Flags.FromLayout, LayoutRef(src, foobar.Tag), dst, "--to-username", Username, "--to-password", Password+"?").
				MatchErrKeyWords(RegistryErrorPrefix).ExpectFailure().Exec()
		})

		It("should fail and show registry error prefix if source registry is not logged in", func() {
			src := RegistryRef(ZOTHost, cpTestRepo("src-not-logged-in"), foobar.Tag)
			dst := RegistryRef(ZOTHost, ArtifactRepo, "")
			ORAS("cp", src, dst, "--from-username", Username, "--from-password", Password+"?").
				MatchErrKeyWords(RegistryErrorPrefix).ExpectFailure().Exec()
		})
	})
})

var foobarStates = append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageConfigStateKey(oras.MediaTypeUnknownConfig))

func CompareRef(src, dst string) {
	srcManifest := ORAS("manifest", "fetch", src).WithDescription("fetch from source to validate").Exec().Out.Contents()
	dstManifest := ORAS("manifest", "fetch", dst).WithDescription("fetch from destination to validate").Exec().Out.Contents()
	Expect(srcManifest).To(Equal(dstManifest))
}

var _ = Describe("1.1 registry users:", func() {
	When("running `cp`", func() {
		It("should copy an artifact with blob", func() {
			src := RegistryRef(ZOTHost, ArtifactRepo, blob.Tag)
			dst := RegistryRef(ZOTHost, cpTestRepo("artifact-with-blob"), "copied")
			ORAS("cp", src, dst, "-v").MatchStatus(blob.StateKeys, true, len(blob.StateKeys)).Exec()
			CompareRef(src, dst)
		})

		It("should copy and show logs with deduplicated id", func() {
			src := RegistryRef(ZOTHost, ArtifactRepo, blob.Tag)
			dst := RegistryRef(ZOTHost, cpTestRepo("debug-log"), "copied")
			session := ORAS("cp", src, dst, "-d").Exec()
			Expect(session.Err).To(gbytes.Say("Request #0"))
			Expect(session.Err).NotTo(gbytes.Say("Request #0"))
			CompareRef(src, dst)
		})

		It("should copy an artifact with config", func() {
			src := RegistryRef(ZOTHost, ArtifactRepo, config.Tag)
			dst := RegistryRef(ZOTHost, cpTestRepo("artifact-with-config"), "copied")
			ORAS("cp", src, dst, "-v").MatchStatus(config.StateKeys, true, len(config.StateKeys)).Exec()
		})

		It("should copy index and its subject", func() {
			stateKeys := append(ma.IndexStateKeys, index.ManifestStatusKey)
			src := RegistryRef(ZOTHost, ArtifactRepo, index.ManifestDigest)
			dst := RegistryRef(ZOTHost, cpTestRepo("index-with-subject"), "")
			ORAS("cp", src, dst, "-v").MatchStatus(stateKeys, true, len(stateKeys)).Exec()
		})

		It("should copy an image to a new repository via tag", func() {
			src := RegistryRef(ZOTHost, ImageRepo, foobar.Tag)
			dst := RegistryRef(ZOTHost, cpTestRepo("tag"), "copied")
			ORAS("cp", src, dst, "-v").MatchStatus(foobarStates, true, len(foobarStates)).Exec()
			CompareRef(src, dst)
		})

		It("should copy an image to a new repository via digest", func() {
			src := RegistryRef(ZOTHost, ImageRepo, foobar.Digest)
			dst := RegistryRef(ZOTHost, cpTestRepo("digest"), "copiedTag")
			ORAS("cp", src, dst, "-v").MatchStatus(foobarStates, true, len(foobarStates)).Exec()
			CompareRef(src, dst)
		})

		It("should copy an image to a new repository via tag without tagging", func() {
			src := RegistryRef(ZOTHost, ImageRepo, foobar.Tag)
			dst := RegistryRef(ZOTHost, cpTestRepo("no-tagging"), foobar.Digest)
			ORAS("cp", src, dst, "-v").MatchStatus(foobarStates, true, len(foobarStates)).Exec()
			CompareRef(src, dst)
		})

		It("should copy an image and its referrers to a new repository", func() {
			stateKeys := append(append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageReferrerConfigStateKeys[0]), foobar.ImageReferrersStateKeys...)
			src := RegistryRef(ZOTHost, ArtifactRepo, foobar.Tag)
			dst := RegistryRef(ZOTHost, cpTestRepo("referrers"), foobar.Digest)
			ORAS("cp", "-r", src, dst, "-v").MatchStatus(stateKeys, true, len(stateKeys)).Exec()
			CompareRef(src, dst)
		})

		It("should copy a multi-arch image and its referrers to a new repository via tag", func() {
			stateKeys := append(ma.IndexStateKeys, ma.IndexZOTReferrerStateKey, ma.LinuxAMD64ReferrerConfigStateKey)
			src := RegistryRef(ZOTHost, ArtifactRepo, ma.Tag)
			dstRepo := cpTestRepo("index-referrers")
			dst := RegistryRef(ZOTHost, dstRepo, "copiedTag")
			ORAS("cp", src, dst, "-r", "-v").
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("Digest: " + ma.Digest).
				Exec()
			// validate
			CompareRef(RegistryRef(ZOTHost, ImageRepo, ma.Digest), dst)
			var index ocispec.Index
			bytes := ORAS("discover", dst, "-o", "json", "--artifact-type", ma.IndexReferrerConfigStateKey.Name).
				MatchKeyWords(ma.IndexReferrerDigest).
				WithDescription("copy image referrer").
				Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].Digest.String()).To(Equal(ma.IndexReferrerDigest))
			ORAS("manifest", "fetch", RegistryRef(ZOTHost, dstRepo, ma.LinuxAMD64Referrer.Digest.String())).
				WithDescription("copy referrer of successor").
				Exec()
		})

		It("should copy a multi-arch image and its referrers without concurrency limitation", func() {
			stateKeys := append(ma.IndexStateKeys, ma.IndexZOTReferrerStateKey, ma.LinuxAMD64ReferrerConfigStateKey)
			src := RegistryRef(ZOTHost, ArtifactRepo, ma.Tag)
			dstRepo := cpTestRepo("index-referrers-concurrent")
			dst := RegistryRef(ZOTHost, dstRepo, "copiedTag")
			// test
			ORAS("cp", src, dst, "-r", "-v", "--concurrency", "0").
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("Digest: " + ma.Digest).
				Exec()
			// validate
			CompareRef(RegistryRef(ZOTHost, ImageRepo, ma.Digest), dst)
			var index ocispec.Index
			bytes := ORAS("discover", dst, "-o", "json", "--artifact-type", ma.IndexReferrerConfigStateKey.Name).
				MatchKeyWords(ma.IndexReferrerDigest).
				WithDescription("copy image referrer").
				Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].Digest.String()).To(Equal(ma.IndexReferrerDigest))
			ORAS("manifest", "fetch", RegistryRef(ZOTHost, dstRepo, ma.LinuxAMD64Referrer.Digest.String())).
				WithDescription("copy referrer of successor").
				Exec()
		})

		It("should copy an empty index", func() {
			src := RegistryRef(ZOTHost, ImageRepo, ma.EmptyTag)
			dstRepo := cpTestRepo("empty-index")
			dst := RegistryRef(ZOTHost, dstRepo, "copiedTag")
			// test
			ORAS("cp", src, dst, "-r", "-v", "--concurrency", "0").Exec()
			// validate
			CompareRef(RegistryRef(ZOTHost, ImageRepo, ma.EmptyTag), dst)
		})

		It("should copy a multi-arch image and its referrers to a new repository via digest", func() {
			stateKeys := append(ma.IndexStateKeys, ma.IndexZOTReferrerStateKey, ma.LinuxAMD64ReferrerConfigStateKey)
			src := RegistryRef(ZOTHost, ArtifactRepo, ma.Tag)
			dstRepo := cpTestRepo("index-referrers-digest")
			dst := RegistryRef(ZOTHost, dstRepo, ma.Digest)
			ORAS("cp", src, dst, "-r", "-v").
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("Digest: " + ma.Digest).
				Exec()
			// validate
			CompareRef(RegistryRef(ZOTHost, ImageRepo, ma.Digest), dst)
			var index ocispec.Index
			bytes := ORAS("discover", dst, "-o", "json", "--artifact-type", ma.IndexReferrerConfigStateKey.Name).
				MatchKeyWords(ma.IndexReferrerDigest).
				WithDescription("copy image referrer").
				Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].Digest.String()).To(Equal(ma.IndexReferrerDigest))
			ORAS("manifest", "fetch", RegistryRef(ZOTHost, dstRepo, ma.LinuxAMD64Referrer.Digest.String())).
				WithDescription("not copy referrer of successor").
				Exec()
		})

		It("should copy a certain platform of image to a new repository via tag", func() {
			src := RegistryRef(ZOTHost, ImageRepo, ma.Tag)
			dst := RegistryRef(ZOTHost, cpTestRepo("platform-tag"), "copiedTag")

			ORAS("cp", src, dst, "--platform", "linux/amd64", "-v").
				MatchStatus(ma.LinuxAMD64StateKeys, true, len(ma.LinuxAMD64StateKeys)).
				MatchKeyWords("Digest: " + ma.LinuxAMD64.Digest.String()).
				Exec()
			CompareRef(RegistryRef(ZOTHost, ImageRepo, ma.LinuxAMD64.Digest.String()), dst)
		})

		It("should copy a certain platform of image to a new repository via digest", func() {
			src := RegistryRef(ZOTHost, ImageRepo, ma.Digest)
			dstRepo := cpTestRepo("platform-digest")
			dst := RegistryRef(ZOTHost, dstRepo, "")
			ORAS("cp", src, dst, "--platform", "linux/amd64", "-v").
				MatchStatus(ma.LinuxAMD64StateKeys, true, len(ma.LinuxAMD64StateKeys)).
				MatchKeyWords("Digest: " + ma.LinuxAMD64.Digest.String()).
				Exec()
			CompareRef(RegistryRef(ZOTHost, ImageRepo, ma.LinuxAMD64.Digest.String()), RegistryRef(ZOTHost, dstRepo, ma.LinuxAMD64.Digest.String()))
		})

		It("should copy a certain platform of image and its referrers to a new repository with tag", func() {
			stateKeys := append(ma.LinuxAMD64StateKeys, ma.LinuxAMD64ReferrerStateKey, ma.LinuxAMD64ReferrerConfigStateKey)
			src := RegistryRef(ZOTHost, ArtifactRepo, ma.Tag)
			dstRepo := cpTestRepo("platform-referrers")
			dst := RegistryRef(ZOTHost, dstRepo, "copiedTag")
			ORAS("cp", src, dst, "-r", "--platform", "linux/amd64", "-v").
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("Digest: " + ma.LinuxAMD64.Digest.String()).
				Exec()
			// validate
			CompareRef(RegistryRef(ZOTHost, ImageRepo, ma.LinuxAMD64.Digest.String()), dst)
			var index ocispec.Index
			bytes := ORAS("discover", dst, "-o", "json", "--platform", "linux/amd64").
				MatchKeyWords(ma.LinuxAMD64Referrer.Digest.String()).
				WithDescription("discover amd64 referrers").
				Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].Digest.String()).To(Equal(ma.LinuxAMD64Referrer.Digest.String()))
			ORAS("manifest", "fetch", RegistryRef(ZOTHost, dstRepo, ma.Digest)).
				WithDescription("not copy index").
				ExpectFailure().
				Exec()
			ORAS("manifest", "fetch", RegistryRef(ZOTHost, dstRepo, ma.IndexReferrerDigest)).
				WithDescription("not copy index referrer").
				ExpectFailure().
				Exec()
		})

		It("should copy a certain platform of image and its referrers to a new repository without tagging", func() {
			stateKeys := append(ma.LinuxAMD64StateKeys, ma.LinuxAMD64ReferrerStateKey, ma.LinuxAMD64ReferrerConfigStateKey)
			src := RegistryRef(ZOTHost, ArtifactRepo, ma.Tag)
			dstRepo := cpTestRepo("platform-referrers-no-tag")
			ORAS("cp", src, RegistryRef(ZOTHost, dstRepo, ""), "-r", "--platform", "linux/amd64", "-v").
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("Digest: " + ma.LinuxAMD64.Digest.String()).
				Exec()
			// validate
			dstRef := RegistryRef(ZOTHost, dstRepo, ma.LinuxAMD64.Digest.String())
			CompareRef(RegistryRef(ZOTHost, ImageRepo, ma.LinuxAMD64.Digest.String()), dstRef)
			var index ocispec.Index
			bytes := ORAS("discover", dstRef, "-o", "json", "--platform", "linux/amd64").
				MatchKeyWords(ma.LinuxAMD64Referrer.Digest.String()).
				WithDescription("discover amd64 referrers").
				Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].Digest.String()).To(Equal(ma.LinuxAMD64Referrer.Digest.String()))
			ORAS("manifest", "fetch", RegistryRef(ZOTHost, dstRepo, ma.Digest)).
				WithDescription("not copy index").
				ExpectFailure().
				Exec()
			ORAS("manifest", "fetch", RegistryRef(ZOTHost, dstRepo, ma.IndexReferrerDigest)).
				WithDescription("not copy index referrer").
				ExpectFailure().
				Exec()
		})

		It("should copy an image to a new repository with multiple tagging", func() {
			src := RegistryRef(ZOTHost, ImageRepo, foobar.Digest)
			tags := []string{"tag1", "tag2", "tag3"}
			dstRepo := cpTestRepo("multi-tagging")
			dst := RegistryRef(ZOTHost, dstRepo, "")
			ORAS("cp", src, dst+":"+strings.Join(tags, ","), "-v").MatchStatus(foobarStates, true, len(foobarStates)).Exec()
			for _, tag := range tags {
				dst := RegistryRef(ZOTHost, dstRepo, tag)
				CompareRef(src, dst)
			}
		})
	})
})

var _ = Describe("OCI spec 1.0 registry users:", func() {
	When("running `cp`", func() {
		It("should copy an image artifact with mounting", func() {
			repo := cpTestRepo("1.0-mount")
			src := RegistryRef(FallbackHost, ArtifactRepo, foobar.Tag)
			dst := RegistryRef(FallbackHost, repo, "")
			out := ORAS("cp", src, dst, "-v").Exec()
			Expect(out).Should(gbytes.Say("Mounted fcde2b2edba5 bar"))
			CompareRef(src, RegistryRef(FallbackHost, repo, foobar.Digest))
		})

		It("should copy an image artifact and its referrers from a registry to a fallback registry", func() {
			repo := cpTestRepo("to-fallback")
			stateKeys := append(append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageReferrerConfigStateKeys[0]), foobar.ImageReferrersStateKeys...)
			src := RegistryRef(ZOTHost, ArtifactRepo, foobar.SignatureImageReferrer.Digest.String())
			dst := RegistryRef(FallbackHost, repo, "")
			ORAS("cp", "-r", src, dst, "-v").MatchStatus(stateKeys, true, len(stateKeys)).Exec()
			CompareRef(src, RegistryRef(FallbackHost, repo, foobar.SignatureImageReferrer.Digest.String()))
			ORAS("discover", "-o", "tree", RegistryRef(FallbackHost, repo, foobar.Digest)).
				WithDescription("discover referrer via subject").MatchKeyWords(foobar.SignatureImageReferrer.Digest.String(), foobar.SBOMImageReferrer.Digest.String()).Exec()
		})
		It("should copy an image artifact and its referrers from a fallback registry to a registry", func() {
			repo := cpTestRepo("from-fallback")
			stateKeys := append(append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageReferrerConfigStateKeys[0]), foobar.ImageReferrersStateKeys...)
			src := RegistryRef(FallbackHost, ArtifactRepo, foobar.SBOMImageReferrer.Digest.String())
			dst := RegistryRef(ZOTHost, repo, "")
			ORAS("cp", "-r", src, dst, "-v").MatchStatus(stateKeys, true, len(stateKeys)).Exec()
			CompareRef(src, RegistryRef(ZOTHost, repo, foobar.SBOMImageReferrer.Digest.String()))
			ORAS("discover", "-o", "tree", RegistryRef(ZOTHost, repo, foobar.Digest)).
				WithDescription("discover referrer via subject").MatchKeyWords(foobar.SignatureImageReferrer.Digest.String(), foobar.SBOMImageReferrer.Digest.String()).Exec()
		})

		It("should copy an image from a fallback registry to an OCI image layout via digest", func() {
			dstDir := GinkgoT().TempDir()
			src := RegistryRef(FallbackHost, ArtifactRepo, foobar.Tag)
			ORAS("cp", src, dstDir, "-v", Flags.ToLayout).MatchStatus(foobarStates, true, len(foobarStates)).Exec()
			// validate
			srcManifest := ORAS("manifest", "fetch", src).WithDescription("fetch from source to validate").Exec().Out.Contents()
			dstManifest := ORAS("manifest", "fetch", LayoutRef(dstDir, foobar.Digest), Flags.Layout).WithDescription("fetch from destination to validate").Exec().Out.Contents()
			Expect(srcManifest).To(Equal(dstManifest))
		})

		It("should copy an image from an OCI image layout to a fallback registry via digest", func() {
			layoutDir := GinkgoT().TempDir()
			src := LayoutRef(layoutDir, foobar.Digest)
			dst := RegistryRef(FallbackHost, cpTestRepo("from-layout-digest"), "copied")
			// prepare
			ORAS("cp", RegistryRef(FallbackHost, ArtifactRepo, foobar.Tag), layoutDir, Flags.ToLayout).Exec()
			// test
			ORAS("cp", src, dst, "-v", Flags.FromLayout).MatchStatus(foobarStates, true, len(foobarStates)).Exec()
			// validate
			srcManifest := ORAS("manifest", "fetch", src, Flags.Layout).WithDescription("fetch from source to validate").Exec().Out.Contents()
			dstManifest := ORAS("manifest", "fetch", dst).WithDescription("fetch from destination to validate").Exec().Out.Contents()
			Expect(srcManifest).To(Equal(dstManifest))
		})

		It("should copy a certain platform of image and its referrers from an OCI image layout to a fallback registry", func() {
			stateKeys := append(ma.LinuxAMD64StateKeys, ma.LinuxAMD64ReferrerStateKey, ma.LinuxAMD64ReferrerConfigStateKey)
			fromDir := GinkgoT().TempDir()
			src := LayoutRef(fromDir, ma.Tag)
			dstRepo := cpTestRepo("platform-referrer-from-layout")
			dst := RegistryRef(FallbackHost, dstRepo, "copied")
			// prepare
			ORAS("cp", RegistryRef(ZOTHost, ArtifactRepo, ma.Tag), src, Flags.ToLayout, "-r").Exec()
			ORAS("cp", RegistryRef(ZOTHost, ArtifactRepo, ma.Tag), src, Flags.ToLayout, "-r", "--platform", "linux/amd64").Exec()
			// test
			ORAS("cp", src, Flags.FromLayout, dst, "-r", "-v", "--platform", "linux/amd64").
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("Digest: " + ma.LinuxAMD64.Digest.String()).
				Exec()
			// validate
			srcManifest := ORAS("manifest", "fetch", src, Flags.Layout, "--platform", "linux/amd64").WithDescription("fetch from source to validate").Exec().Out.Contents()
			dstManifest := ORAS("manifest", "fetch", dst).WithDescription("fetch from destination to validate").Exec().Out.Contents()
			Expect(srcManifest).To(Equal(dstManifest))
			ORAS("manifest", "fetch", RegistryRef(FallbackHost, dstRepo, ma.Digest)).WithDescription("not copy index").ExpectFailure().Exec()
			var index ocispec.Index
			bytes := ORAS("discover", dst, "-o", "json").
				MatchKeyWords(ma.LinuxAMD64Referrer.Digest.String()).
				WithDescription("copy image referrer").
				Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].Digest.String()).To(Equal(ma.LinuxAMD64Referrer.Digest.String()))
		})
	})
})

var _ = Describe("OCI layout users:", func() {
	When("running `cp`", func() {
		It("should copy an image from a registry to an OCI image layout via tag", func() {
			dst := LayoutRef(GinkgoT().TempDir(), "copied")
			src := RegistryRef(ZOTHost, ImageRepo, foobar.Tag)
			ORAS("cp", src, dst, "-v", Flags.ToLayout).MatchStatus(foobarStates, true, len(foobarStates)).Exec()
			// validate
			srcManifest := ORAS("manifest", "fetch", src).WithDescription("fetch from source to validate").Exec().Out.Contents()
			dstManifest := ORAS("manifest", "fetch", dst, Flags.Layout).WithDescription("fetch from destination to validate").Exec().Out.Contents()
			Expect(srcManifest).To(Equal(dstManifest))
		})

		It("should copy an image from an OCI image layout to a registry via tag", func() {
			layoutDir := GinkgoT().TempDir()
			src := LayoutRef(layoutDir, "copied")
			dst := RegistryRef(ZOTHost, cpTestRepo("from-layout-tag"), foobar.Tag)
			// prepare
			ORAS("cp", RegistryRef(ZOTHost, ImageRepo, foobar.Tag), src, Flags.ToLayout).Exec()
			// test
			ORAS("cp", src, dst, "-v", Flags.FromLayout).MatchStatus(foobarStates, true, len(foobarStates)).Exec()
			// validate
			srcManifest := ORAS("manifest", "fetch", src, Flags.Layout).WithDescription("fetch from source to validate").Exec().Out.Contents()
			dstManifest := ORAS("manifest", "fetch", dst).WithDescription("fetch from destination to validate").Exec().Out.Contents()
			Expect(srcManifest).To(Equal(dstManifest))
		})

		It("should copy an image between OCI image layouts via tag", func() {
			srcDir := GinkgoT().TempDir()
			toDir := GinkgoT().TempDir()
			src := LayoutRef(srcDir, "from")
			dst := LayoutRef(toDir, "to")
			// prepare
			ORAS("cp", RegistryRef(ZOTHost, ImageRepo, foobar.Tag), src, Flags.ToLayout).Exec()
			// test
			ORAS("cp", src, dst, "-v", Flags.FromLayout, Flags.ToLayout).MatchStatus(foobarStates, true, len(foobarStates)).Exec()
			// validate
			srcManifest := ORAS("manifest", "fetch", src, Flags.Layout).WithDescription("fetch from source to validate").Exec().Out.Contents()
			dstManifest := ORAS("manifest", "fetch", dst, Flags.Layout).WithDescription("fetch from destination to validate").Exec().Out.Contents()
			Expect(srcManifest).To(Equal(dstManifest))
		})

		It("should copy an image from a registry to an OCI image layout via digest", func() {
			dstDir := GinkgoT().TempDir()
			src := RegistryRef(ZOTHost, ImageRepo, foobar.Tag)
			ORAS("cp", src, dstDir, "-v", Flags.ToLayout).MatchStatus(foobarStates, true, len(foobarStates)).Exec()
			// validate
			srcManifest := ORAS("manifest", "fetch", src).WithDescription("fetch from source to validate").Exec().Out.Contents()
			dstManifest := ORAS("manifest", "fetch", LayoutRef(dstDir, foobar.Digest), Flags.Layout).WithDescription("fetch from destination to validate").Exec().Out.Contents()
			Expect(srcManifest).To(Equal(dstManifest))
		})

		It("should copy an image from an OCI image layout to a registry via digest", func() {
			layoutDir := GinkgoT().TempDir()
			src := LayoutRef(layoutDir, foobar.Digest)
			dst := RegistryRef(ZOTHost, cpTestRepo("from-layout-digest"), "copied")
			// prepare
			ORAS("cp", RegistryRef(ZOTHost, ImageRepo, foobar.Tag), layoutDir, Flags.ToLayout).Exec()
			// test
			ORAS("cp", src, dst, "-v", Flags.FromLayout).MatchStatus(foobarStates, true, len(foobarStates)).Exec()
			// validate
			srcManifest := ORAS("manifest", "fetch", src, Flags.Layout).WithDescription("fetch from source to validate").Exec().Out.Contents()
			dstManifest := ORAS("manifest", "fetch", dst).WithDescription("fetch from destination to validate").Exec().Out.Contents()
			Expect(srcManifest).To(Equal(dstManifest))
		})

		It("should copy an image between OCI image layouts via digest", func() {
			srcDir := GinkgoT().TempDir()
			toDir := GinkgoT().TempDir()
			src := LayoutRef(srcDir, foobar.Digest)
			dst := LayoutRef(toDir, foobar.Digest)
			// prepare
			ORAS("cp", RegistryRef(ZOTHost, ImageRepo, foobar.Tag), srcDir, Flags.ToLayout).Exec()
			// test
			ORAS("cp", src, toDir, "-v", Flags.FromLayout, Flags.ToLayout).MatchStatus(foobarStates, true, len(foobarStates)).Exec()
			// validate
			srcManifest := ORAS("manifest", "fetch", src, Flags.Layout).WithDescription("fetch from source to validate").Exec().Out.Contents()
			dstManifest := ORAS("manifest", "fetch", dst, Flags.Layout).WithDescription("fetch from destination to validate").Exec().Out.Contents()
			Expect(srcManifest).To(Equal(dstManifest))
		})

		It("should copy an image from a registry to an OCI image layout with multiple tagging", func() {
			dstDir := GinkgoT().TempDir()
			src := RegistryRef(ZOTHost, ImageRepo, foobar.Tag)
			tags := []string{"tag1", "tag2", "tag3"}
			// test
			ORAS("cp", src, dstDir+":"+strings.Join(tags, ","), "-v", Flags.ToLayout).MatchStatus(foobarStates, true, len(foobarStates)).Exec()
			// validate
			srcManifest := ORAS("manifest", "fetch", src).WithDescription("fetch from source to validate").Exec().Out.Contents()
			for _, tag := range tags {
				dstManifest := ORAS("manifest", "fetch", LayoutRef(dstDir, tag), Flags.Layout).WithDescription("fetch from destination to validate").Exec().Out.Contents()
				Expect(srcManifest).To(Equal(dstManifest))
			}
		})

		It("should copy a tagged image and its referrers from a registry to an OCI image layout", func() {
			stateKeys := append(append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageReferrerConfigStateKeys[0]), foobar.ImageReferrersStateKeys...)
			dst := LayoutRef(GinkgoT().TempDir(), "copied")
			src := RegistryRef(ZOTHost, ArtifactRepo, foobar.Tag)
			// test
			ORAS("cp", "-r", src, dst, "-v", Flags.ToLayout).MatchStatus(stateKeys, true, len(stateKeys)).Exec()
			// validate
			srcManifest := ORAS("manifest", "fetch", src).WithDescription("fetch from source to validate").Exec().Out.Contents()
			dstManifest := ORAS("manifest", "fetch", dst, Flags.Layout).WithDescription("fetch from destination to validate").Exec().Out.Contents()
			Expect(srcManifest).To(Equal(dstManifest))
		})

		It("should copy a image and its referrers from a registry to an OCI image layout via digest", func() {
			stateKeys := append(append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageReferrerConfigStateKeys[0]), foobar.ImageReferrersStateKeys...)
			toDir := GinkgoT().TempDir()
			src := RegistryRef(ZOTHost, ArtifactRepo, foobar.Digest)
			// test
			ORAS("cp", "-r", src, toDir, "-v", Flags.ToLayout).MatchStatus(stateKeys, true, len(stateKeys)).Exec()
			// validate
			srcManifest := ORAS("manifest", "fetch", src).WithDescription("fetch from source to validate").Exec().Out.Contents()
			dstManifest := ORAS("manifest", "fetch", LayoutRef(toDir, foobar.Digest), Flags.Layout).WithDescription("fetch from destination to validate").Exec().Out.Contents()
			Expect(srcManifest).To(Equal(dstManifest))
		})

		It("should copy a multi-arch image and its referrers from a registry to an OCI image layout a via tag", func() {
			stateKeys := append(ma.IndexStateKeys, ma.IndexZOTReferrerStateKey, ma.LinuxAMD64ReferrerConfigStateKey)
			src := RegistryRef(ZOTHost, ArtifactRepo, ma.Tag)
			toDir := GinkgoT().TempDir()
			dst := LayoutRef(toDir, "copied")
			// test
			ORAS("cp", src, Flags.ToLayout, dst, "-r", "-v").
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("Digest: " + ma.Digest).
				Exec()
			// validate
			srcManifest := ORAS("manifest", "fetch", src).WithDescription("fetch from source to validate").Exec().Out.Contents()
			dstManifest := ORAS("manifest", "fetch", dst, Flags.Layout).WithDescription("fetch from destination to validate").Exec().Out.Contents()
			Expect(srcManifest).To(Equal(dstManifest))
			var index ocispec.Index
			bytes := ORAS("discover", dst, "-o", "json", Flags.Layout, "--artifact-type", ma.IndexReferrerConfigStateKey.Name).
				// MatchKeyWords(ma.IndexReferrerDigest).
				WithDescription("copy image referrer").
				Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].Digest.String()).To(Equal(ma.IndexReferrerDigest))
			ORAS("manifest", "fetch", Flags.Layout, LayoutRef(toDir, ma.LinuxAMD64Referrer.Digest.String())).
				WithDescription("copy referrer of successor").
				Exec()
		})

		It("should copy a multi-arch image and its referrers from an OCI image layout to a registry via digest", func() {
			stateKeys := append(ma.IndexStateKeys, ma.IndexZOTReferrerStateKey, ma.LinuxAMD64ReferrerConfigStateKey)
			fromDir := GinkgoT().TempDir()
			src := LayoutRef(fromDir, ma.Tag)
			dst := RegistryRef(ZOTHost, cpTestRepo("recursive-from-layout"), "copied")
			// prepare
			ORAS("cp", RegistryRef(ZOTHost, ArtifactRepo, ma.Tag), src, Flags.ToLayout, "-r").Exec()
			// test
			ORAS("cp", src, Flags.FromLayout, dst, "-r", "-v").
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("Digest: " + ma.Digest).
				Exec()
			// validate
			srcManifest := ORAS("manifest", "fetch", src, Flags.Layout).WithDescription("fetch from source to validate").Exec().Out.Contents()
			dstManifest := ORAS("manifest", "fetch", dst).WithDescription("fetch from destination to validate").Exec().Out.Contents()
			Expect(srcManifest).To(Equal(dstManifest))
			var index ocispec.Index
			bytes := ORAS("discover", dst, "-o", "json", "--artifact-type", ma.IndexReferrerConfigStateKey.Name).
				MatchKeyWords(ma.IndexReferrerDigest).
				WithDescription("copy image referrer").
				Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].Digest.String()).To(Equal(ma.IndexReferrerDigest))
			ORAS("manifest", "fetch", dst).
				WithDescription("copy referrer of successor").
				Exec()
		})

		It("should copy a certain platform of image and its referrers from an OCI image layout to a registry", func() {
			stateKeys := append(ma.LinuxAMD64StateKeys, ma.LinuxAMD64ReferrerStateKey, ma.LinuxAMD64ReferrerConfigStateKey)
			fromDir := GinkgoT().TempDir()
			src := LayoutRef(fromDir, ma.Tag)
			dstRepo := cpTestRepo("platform-referrer-from-layout")
			dst := RegistryRef(ZOTHost, dstRepo, "copied")
			// prepare
			ORAS("cp", RegistryRef(ZOTHost, ArtifactRepo, ma.Tag), src, Flags.ToLayout, "-r").Exec()
			ORAS("cp", RegistryRef(ZOTHost, ArtifactRepo, ma.Tag), src, Flags.ToLayout, "-r", "--platform", "linux/amd64").Exec()
			// test
			ORAS("cp", src, Flags.FromLayout, dst, "-r", "-v", "--platform", "linux/amd64").
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("Digest: " + ma.LinuxAMD64.Digest.String()).
				Exec()
			// validate
			srcManifest := ORAS("manifest", "fetch", src, Flags.Layout, "--platform", "linux/amd64").WithDescription("fetch from source to validate").Exec().Out.Contents()
			dstManifest := ORAS("manifest", "fetch", dst).WithDescription("fetch from destination to validate").Exec().Out.Contents()
			Expect(srcManifest).To(Equal(dstManifest))
			ORAS("manifest", "fetch", RegistryRef(ZOTHost, dstRepo, ma.Digest)).WithDescription("not copy index").ExpectFailure().Exec()
			var index ocispec.Index
			bytes := ORAS("discover", dst, "-o", "json").
				MatchKeyWords(ma.LinuxAMD64Referrer.Digest.String()).
				WithDescription("copy image referrer").
				Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].Digest.String()).To(Equal(ma.LinuxAMD64Referrer.Digest.String()))
		})

		It("should copy a certain platform of image and its referrers from a registry to an OCI image layout", func() {
			stateKeys := append(ma.LinuxAMD64StateKeys, ma.LinuxAMD64ReferrerStateKey, ma.LinuxAMD64ReferrerConfigStateKey)
			src := RegistryRef(ZOTHost, ArtifactRepo, ma.Tag)
			toDir := GinkgoT().TempDir()
			dst := LayoutRef(toDir, "copied")
			// test
			ORAS("cp", src, Flags.ToLayout, dst, "-r", "-v", "--platform", "linux/amd64").
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("Digest: " + ma.LinuxAMD64.Digest.String()).
				Exec()
			// validate
			srcManifest := ORAS("manifest", "fetch", src, "--platform", "linux/amd64").WithDescription("fetch from source to validate").Exec().Out.Contents()
			dstManifest := ORAS("manifest", "fetch", dst, Flags.Layout).WithDescription("fetch from destination to validate").Exec().Out.Contents()
			Expect(srcManifest).To(Equal(dstManifest))
			ORAS("manifest", "fetch", LayoutRef(toDir, ma.Digest)).WithDescription("not copy index").ExpectFailure().Exec()
			var index ocispec.Index
			bytes := ORAS("discover", dst, "-o", "json", Flags.Layout).
				MatchKeyWords(ma.LinuxAMD64Referrer.Digest.String()).
				WithDescription("copy image referrer").
				Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].Digest.String()).To(Equal(ma.LinuxAMD64Referrer.Digest.String()))
		})
	})
})

var _ = Describe("OCI image spec v1.1.0-rc2 artifact users:", func() {
	It("should copy an image and its referrers to a new repository", func() {
		stateKeys := append(foobarStates, foobar.ArtifactReferrerStateKeys...)
		digest := foobar.SBOMArtifactReferrer.Digest.String()
		src := RegistryRef(Host, ArtifactRepo, digest)
		dst := RegistryRef(Host, cpTestRepo("referrers"), digest)
		ORAS("cp", "-r", src, dst, "-v").MatchStatus(stateKeys, true, len(stateKeys)).Exec()
		CompareRef(src, dst)
	})
})
