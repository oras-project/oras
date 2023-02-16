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
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
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
			ORAS("cp", Reference(Host, ImageRepo, foobar.Tag)).ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})

		It("should fail when source doesn't exist", func() {
			ORAS("cp", Reference(Host, ImageRepo, "i-dont-think-this-tag-exists"), Reference(Host, cpTestRepo("nonexistent-source"), "")).ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})
	})
})

var foobarStates = append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.FileConfigStateKey)

var _ = Describe("Common registry users:", func() {
	When("running `cp`", func() {
		validate := func(src, dst string) {
			srcManifest := ORAS("manifest", "fetch", src).WithDescription("fetch from source for validation").Exec().Out.Contents()
			dstManifest := ORAS("manifest", "fetch", dst).WithDescription("fetch from destination for validation").Exec().Out.Contents()
			Expect(srcManifest).To(Equal(dstManifest))
		}
		It("should copy an image to a new repository via tag", func() {
			src := Reference(Host, ImageRepo, foobar.Tag)
			dst := Reference(Host, cpTestRepo("tag"), "copiedTag")
			ORAS("cp", src, dst, "-v").MatchStatus(foobarStates, true, len(foobarStates)).Exec()
			validate(src, dst)
		})

		It("should copy an image to a new repository via digest", func() {
			src := Reference(Host, ImageRepo, foobar.Digest)
			dst := Reference(Host, cpTestRepo("digest"), "copiedTag")
			ORAS("cp", src, dst, "-v").MatchStatus(foobarStates, true, len(foobarStates)).Exec()
			validate(src, dst)
		})

		It("should copy an image to a new repository via tag without tagging", func() {
			src := Reference(Host, ImageRepo, foobar.Tag)
			dst := Reference(Host, cpTestRepo("no-tagging"), foobar.Digest)
			ORAS("cp", src, dst, "-v").MatchStatus(foobarStates, true, len(foobarStates)).Exec()
			validate(src, dst)
		})

		It("should copy an image and its referrers to a new repository", func() {
			stateKeys := append(append(foobarStates, foobar.ArtifactReferrerStateKeys...), foobar.ImageReferrerConfigStateKeys...)
			src := Reference(Host, ArtifactRepo, foobar.Tag)
			dst := Reference(Host, cpTestRepo("referrers"), foobar.Digest)
			ORAS("cp", "-r", src, dst, "-v").MatchStatus(stateKeys, true, len(stateKeys)).Exec()
			validate(src, dst)
		})

		It("should copy a certain platform of image to a new repository via tag", func() {
			src := Reference(Host, ImageRepo, ma.Tag)
			dst := Reference(Host, cpTestRepo("platform-tag"), "copiedTag")
			stateKeys := append(ma.ImageStates, ma.LinuxAMD64ReferrerState, ma.LinuxAMD64ReferrerConfigState)

			ORAS("cp", src, dst, "--platform", "linux/amd64", "-v").
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("Digest: " + ma.LinuxAMD64Digest).
				Exec()
			validate(Reference(Host, ImageRepo, ma.LinuxAMD64Digest), dst)
		})

		It("should copy a certain platform of image to a new repository via digest", func() {
			src := Reference(Host, ImageRepo, ma.Digest)
			dst := Reference(Host, cpTestRepo("platform-digest"), "copiedTag")
			stateKeys := append(ma.ImageStates, ma.LinuxAMD64ReferrerState, ma.LinuxAMD64ReferrerConfigState)
			ORAS("cp", src, dst, "--platform", "linux/amd64", "-v").
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("Digest: " + ma.LinuxAMD64Digest).
				Exec()
			validate(Reference(Host, ImageRepo, ma.LinuxAMD64Digest), dst)
		})

		It("should copy a certain platform of image and its referrers to a new repository via tag", func() {
			src := Reference(Host, ArtifactRepo, ma.Tag)
			dstRepo := cpTestRepo("platform-referrers-index")
			dst := Reference(Host, dstRepo, "copiedTag")
			ORAS("cp", src, dst, "-r", "--platform", "linux/amd64", "-v").
				MatchStatus(ma.ImageStates, true, len(ma.ImageStates)).
				MatchKeyWords("Digest: " + ma.LinuxAMD64Digest).
				Exec()
			// validate
			validate(Reference(Host, ImageRepo, ma.LinuxAMD64Digest), dst)
			var index ocispec.Index
			bytes := ORAS("discover", dst, "-o", "json", "--platform", "linux/amd64").
				MatchKeyWords(ma.LinuxAMD64ReferrerDigest).Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].Digest.String()).To(Equal(ma.LinuxAMD64ReferrerDigest))
		})

		It("should copy an image to a new repository with multiple tagging", func() {
			src := Reference(Host, ImageRepo, foobar.Digest)
			tags := []string{"tag1", "tag2", "tag3"}
			dstRepo := cpTestRepo("multi-tagging")
			dst := Reference(Host, dstRepo, "")
			ORAS("cp", src, dst+":"+strings.Join(tags, ","), "-v").MatchStatus(foobarStates, true, len(foobarStates)).Exec()
			for _, tag := range tags {
				dst := Reference(Host, dstRepo, tag)
				validate(src, dst)
			}
		})
	})
})

var _ = Describe("OCI spec 1.0 registry users:", func() {
	When("running `cp`", func() {
		validate := func(src, dst string) {
			srcManifest := ORAS("manifest", "fetch", src).Exec().Out.Contents()
			dstManifest := ORAS("manifest", "fetch", dst).Exec().Out.Contents()
			gomega.Expect(srcManifest).To(gomega.Equal(dstManifest))
		}
		It("should copy an image artifact and its referrers from a registry to a fallback registry", func() {
			repo := cpTestRepo("to-fallback")
			stateKeys := append(append(foobarStates, foobar.ImageReferrersStateKeys...), foobar.ImageReferrerConfigStateKeys...)
			src := Reference(Host, ArtifactRepo, foobar.SignatureImageReferrerDigest)
			dst := Reference(FallbackHost, repo, "")
			ORAS("cp", "-r", src, dst, "-v").MatchStatus(stateKeys, true, len(stateKeys)).Exec()
			validate(src, Reference(FallbackHost, repo, foobar.SignatureImageReferrerDigest))
			ORAS("discover", "-o", "tree", Reference(FallbackHost, repo, foobar.Digest)).
				WithDescription("discover referrer via subject").MatchKeyWords(foobar.SignatureImageReferrerDigest, foobar.SBOMImageReferrerDigest).Exec()
		})
		It("should copy an image artifact and its referrers from a fallback registry to a registry", func() {
			repo := cpTestRepo("from-fallback")
			stateKeys := append(append(foobarStates, foobar.FallbackImageReferrersStateKeys...), foobar.ImageReferrerConfigStateKeys...)
			src := Reference(FallbackHost, ArtifactRepo, foobar.FallbackSBOMImageReferrerDigest)
			dst := Reference(Host, repo, "")
			ORAS("cp", "-r", src, dst, "-v").MatchStatus(stateKeys, true, len(stateKeys)).Exec()
			validate(src, Reference(Host, repo, foobar.FallbackSBOMImageReferrerDigest))
			ORAS("discover", "-o", "tree", Reference(Host, repo, foobar.Digest)).
				WithDescription("discover referrer via subject").MatchKeyWords(foobar.FallbackSignatureImageReferrerDigest, foobar.FallbackSBOMImageReferrerDigest).Exec()
		})
	})
})
