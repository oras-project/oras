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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	ma "oras.land/oras/test/e2e/internal/testdata/multi_arch"
	. "oras.land/oras/test/e2e/internal/utils"
)

func cpTestRepo(text string) string {
	return fmt.Sprintf("command/copy/%d/%s", GinkgoRandomSeed(), text)
}

var _ = Describe("ORAS beginners:", func() {
	When("running cp command", func() {
		RunAndShowPreviewInHelp([]string{"copy"})

		It("should show preview and help doc", func() {
			ORAS("cp", "--help").MatchKeyWords("[Preview] Copy", PreviewDesc, ExampleDesc).Exec()
		})

		It("should fail when no reference provided", func() {
			ORAS("cp").ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})

		It("should fail when no destination reference provided", func() {
			ORAS("cp", RegistryRef(Host, ImageRepo, foobar.Tag)).ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})

		It("should fail when source doesn't exist", func() {
			ORAS("cp", RegistryRef(Host, ImageRepo, "i-dont-think-this-tag-exists"), RegistryRef(Host, cpTestRepo("nonexistent-source"), "")).ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})
	})
})

var foobarStates = append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageConfigStateKey(oras.MediaTypeUnknownConfig))

func CompareRef(src, dst string) {
	srcManifest := ORAS("manifest", "fetch", src).WithDescription("fetch from source for validation").Exec().Out.Contents()
	dstManifest := ORAS("manifest", "fetch", dst).WithDescription("fetch from destination for validation").Exec().Out.Contents()
	Expect(srcManifest).To(Equal(dstManifest))
}

var _ = Describe("Common registry users:", func() {
	When("running `cp`", func() {

		It("should copy an image to a new repository via tag", func() {
			src := RegistryRef(Host, ImageRepo, foobar.Tag)
			dst := RegistryRef(Host, cpTestRepo("tag"), "copiedTag")
			ORAS("cp", src, dst, "-v").MatchStatus(foobarStates, true, len(foobarStates)).Exec()
			CompareRef(src, dst)
		})

		It("should copy an image to a new repository via digest", func() {
			src := RegistryRef(Host, ImageRepo, foobar.Digest)
			dst := RegistryRef(Host, cpTestRepo("digest"), "copiedTag")
			ORAS("cp", src, dst, "-v").MatchStatus(foobarStates, true, len(foobarStates)).Exec()
			CompareRef(src, dst)
		})

		It("should copy an image to a new repository via tag without tagging", func() {
			src := RegistryRef(Host, ImageRepo, foobar.Tag)
			dst := RegistryRef(Host, cpTestRepo("no-tagging"), foobar.Digest)
			ORAS("cp", src, dst, "-v").MatchStatus(foobarStates, true, len(foobarStates)).Exec()
			CompareRef(src, dst)
		})

		It("should copy an image and its referrers to a new repository", func() {
			stateKeys := append(append(foobarStates, foobar.ArtifactReferrerStateKeys...), foobar.ImageReferrerConfigStateKeys...)
			src := RegistryRef(Host, ArtifactRepo, foobar.Tag)
			dst := RegistryRef(Host, cpTestRepo("referrers"), foobar.Digest)
			ORAS("cp", "-r", src, dst, "-v").MatchStatus(stateKeys, true, len(stateKeys)).Exec()
			CompareRef(src, dst)
		})

		It("should copy a multi-arch image and its referrers to a new repository via tag", func() {
			src := RegistryRef(Host, ArtifactRepo, ma.Tag)
			dstRepo := cpTestRepo("index-referrers")
			dst := RegistryRef(Host, dstRepo, "copiedTag")
			ORAS("cp", src, dst, "-r", "-v").
				MatchStatus(ma.IndexStateKeys, true, len(ma.IndexStateKeys)).
				MatchKeyWords("Digest: " + ma.Digest).
				Exec()
			// validate
			CompareRef(RegistryRef(Host, ImageRepo, ma.Digest), dst)
			var index ocispec.Index
			bytes := ORAS("discover", dst, "-o", "json").
				MatchKeyWords(ma.IndexReferrerDigest).
				WithDescription("copy image referrer").
				Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].Digest.String()).To(Equal(ma.IndexReferrerDigest))
			ORAS("manifest", "fetch", RegistryRef(Host, dstRepo, ma.LinuxAMD64Referrer.Digest.String())).
				WithDescription("not copy referrer of successor").
				ExpectFailure().
				Exec()
		})

		It("should copy a multi-arch image and its referrers to a new repository via digest", func() {
			src := RegistryRef(Host, ArtifactRepo, ma.Tag)
			dstRepo := cpTestRepo("index-referrers-digest")
			dst := RegistryRef(Host, dstRepo, ma.Digest)
			ORAS("cp", src, dst, "-r", "-v").
				MatchStatus(ma.IndexStateKeys, true, len(ma.IndexStateKeys)).
				MatchKeyWords("Digest: " + ma.Digest).
				Exec()
			// validate
			CompareRef(RegistryRef(Host, ImageRepo, ma.Digest), dst)
			var index ocispec.Index
			bytes := ORAS("discover", dst, "-o", "json").
				MatchKeyWords(ma.IndexReferrerDigest).
				WithDescription("copy image referrer").
				Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].Digest.String()).To(Equal(ma.IndexReferrerDigest))
			ORAS("manifest", "fetch", RegistryRef(Host, dstRepo, ma.LinuxAMD64Referrer.Digest.String())).
				WithDescription("not copy referrer of successor").
				ExpectFailure().
				Exec()
		})

		It("should copy a certain platform of image to a new repository via tag", func() {
			src := RegistryRef(Host, ImageRepo, ma.Tag)
			dst := RegistryRef(Host, cpTestRepo("platform-tag"), "copiedTag")

			ORAS("cp", src, dst, "--platform", "linux/amd64", "-v").
				MatchStatus(ma.IndexStateKeys, true, len(ma.IndexStateKeys)).
				MatchKeyWords("Digest: " + ma.LinuxAMD64.Digest.String()).
				Exec()
			CompareRef(RegistryRef(Host, ImageRepo, ma.LinuxAMD64.Digest.String()), dst)
		})

		It("should copy a certain platform of image to a new repository via digest", func() {
			src := RegistryRef(Host, ImageRepo, ma.Digest)
			dstRepo := cpTestRepo("platform-digest")
			dst := RegistryRef(Host, dstRepo, "")
			ORAS("cp", src, dst, "--platform", "linux/amd64", "-v").
				MatchStatus(ma.IndexStateKeys, true, len(ma.IndexStateKeys)).
				MatchKeyWords("Digest: " + ma.LinuxAMD64.Digest.String()).
				Exec()
			CompareRef(RegistryRef(Host, ImageRepo, ma.LinuxAMD64.Digest.String()), RegistryRef(Host, dstRepo, ma.LinuxAMD64.Digest.String()))
		})

		It("should copy a certain platform of image and its referrers to a new repository with tag", func() {
			src := RegistryRef(Host, ArtifactRepo, ma.Tag)
			dstRepo := cpTestRepo("platform-referrers")
			dst := RegistryRef(Host, dstRepo, "copiedTag")
			ORAS("cp", src, dst, "-r", "--platform", "linux/amd64", "-v").
				MatchStatus(ma.IndexStateKeys, true, len(ma.IndexStateKeys)).
				MatchKeyWords("Digest: " + ma.LinuxAMD64.Digest.String()).
				Exec()
			// validate
			CompareRef(RegistryRef(Host, ImageRepo, ma.LinuxAMD64.Digest.String()), dst)
			var index ocispec.Index
			bytes := ORAS("discover", dst, "-o", "json", "--platform", "linux/amd64").
				MatchKeyWords(ma.LinuxAMD64Referrer.Digest.String()).
				WithDescription("discover amd64 referrers").
				Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].Digest.String()).To(Equal(ma.LinuxAMD64Referrer.Digest.String()))
			ORAS("manifest", "fetch", RegistryRef(Host, dstRepo, ma.Digest)).
				WithDescription("not copy index").
				ExpectFailure().
				Exec()
			ORAS("manifest", "fetch", RegistryRef(Host, dstRepo, ma.IndexReferrerDigest)).
				WithDescription("not copy index referrer").
				ExpectFailure().
				Exec()
		})

		It("should copy a certain platform of image and its referrers to a new repository without tagging", func() {
			src := RegistryRef(Host, ArtifactRepo, ma.Tag)
			dstRepo := cpTestRepo("platform-referrers-no-tag")
			ORAS("cp", src, RegistryRef(Host, dstRepo, ""), "-r", "--platform", "linux/amd64", "-v").
				MatchStatus(ma.IndexStateKeys, true, len(ma.IndexStateKeys)).
				MatchKeyWords("Digest: " + ma.LinuxAMD64.Digest.String()).
				Exec()
			// validate
			dstRef := RegistryRef(Host, dstRepo, ma.LinuxAMD64.Digest.String())
			CompareRef(RegistryRef(Host, ImageRepo, ma.LinuxAMD64.Digest.String()), dstRef)
			var index ocispec.Index
			bytes := ORAS("discover", dstRef, "-o", "json", "--platform", "linux/amd64").
				MatchKeyWords(ma.LinuxAMD64Referrer.Digest.String()).
				WithDescription("discover amd64 referrers").
				Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].Digest.String()).To(Equal(ma.LinuxAMD64Referrer.Digest.String()))
			ORAS("manifest", "fetch", RegistryRef(Host, dstRepo, ma.Digest)).
				WithDescription("not copy index").
				ExpectFailure().
				Exec()
			ORAS("manifest", "fetch", RegistryRef(Host, dstRepo, ma.IndexReferrerDigest)).
				WithDescription("not copy index referrer").
				ExpectFailure().
				Exec()
		})

		It("should copy an image to a new repository with multiple tagging", func() {
			src := RegistryRef(Host, ImageRepo, foobar.Digest)
			tags := []string{"tag1", "tag2", "tag3"}
			dstRepo := cpTestRepo("multi-tagging")
			dst := RegistryRef(Host, dstRepo, "")
			ORAS("cp", src, dst+":"+strings.Join(tags, ","), "-v").MatchStatus(foobarStates, true, len(foobarStates)).Exec()
			for _, tag := range tags {
				dst := RegistryRef(Host, dstRepo, tag)
				CompareRef(src, dst)
			}
		})
	})
})

var _ = Describe("OCI spec 1.0 registry users:", func() {
	When("running `cp`", func() {
		It("should copy an image artifact and its referrers from a registry to a fallback registry", func() {
			repo := cpTestRepo("to-fallback")
			stateKeys := append(append(foobarStates, foobar.ImageReferrersStateKeys...), foobar.ImageReferrerConfigStateKeys...)
			src := RegistryRef(Host, ArtifactRepo, foobar.SignatureImageReferrer.Digest.String())
			dst := RegistryRef(FallbackHost, repo, "")
			ORAS("cp", "-r", src, dst, "-v").MatchStatus(stateKeys, true, len(stateKeys)).Exec()
			CompareRef(src, RegistryRef(FallbackHost, repo, foobar.SignatureImageReferrer.Digest.String()))
			ORAS("discover", "-o", "tree", RegistryRef(FallbackHost, repo, foobar.Digest)).
				WithDescription("discover referrer via subject").MatchKeyWords(foobar.SignatureImageReferrer.Digest.String(), foobar.SBOMImageReferrer.Digest.String()).Exec()
		})
		It("should copy an image artifact and its referrers from a fallback registry to a registry", func() {
			repo := cpTestRepo("from-fallback")
			stateKeys := append(append(foobarStates, foobar.FallbackImageReferrersStateKeys...), foobar.ImageReferrerConfigStateKeys...)
			src := RegistryRef(FallbackHost, ArtifactRepo, foobar.FallbackSBOMImageReferrer.Digest.String())
			dst := RegistryRef(Host, repo, "")
			ORAS("cp", "-r", src, dst, "-v").MatchStatus(stateKeys, true, len(stateKeys)).Exec()
			CompareRef(src, RegistryRef(Host, repo, foobar.FallbackSBOMImageReferrer.Digest.String()))
			ORAS("discover", "-o", "tree", RegistryRef(Host, repo, foobar.Digest)).
				WithDescription("discover referrer via subject").MatchKeyWords(foobar.FallbackSignatureImageReferrer.Digest.String(), foobar.FallbackSBOMImageReferrer.Digest.String()).Exec()
		})
	})
})
