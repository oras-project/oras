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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	. "oras.land/oras/test/e2e/internal/utils"
	"oras.land/oras/test/e2e/internal/utils/match"
)

func cpRepo(text string) string {
	return fmt.Sprintf("command/copy/%d/%s", GinkgoRandomSeed(), text)
}

var _ = Describe("ORAS beginners:", func() {
	When("running repo command", func() {
		RunAndShowPreviewInHelp([]string{"copy"})

		It("should show preview and help doc", func() {
			ORAS("cp", "--help").MatchKeyWords("[Preview] Copy", PreviewDesc, ExampleDesc).Exec()
		})

		It("should fail when no reference provided", func() {
			ORAS("cp").ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})

		It("should fail when no destination reference provided", func() {
			ORAS("cp", Reference(Host, Repo, FoobarImageTag)).ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})

		It("should fail when source doesn't exist", func() {
			ORAS("cp", Reference(Host, Repo, "i-dont-think-this-tag-exists"), Reference(Host, cpRepo("nonexistent-source"), "")).ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})
	})
})

var _ = Describe("Common registry users:", func() {
	When("running `cp`", func() {
		imageStateKeys := []match.StateKey{
			{Digest: "44136fa355b3", Name: "application/vnd.unknown.config.v1+json"},
			{Digest: "fcde2b2edba5", Name: "bar"},
			{Digest: "2c26b46b68ff", Name: "foo1"},
			{Digest: "2c26b46b68ff", Name: "foo2"},
			{Digest: "fd6ed2f36b54", Name: "application/vnd.oci.image.manifest.v1+json"},
		}
		validate := func(src, dst string) {
			srcManifest := ORAS("manifest", "fetch", src).Exec().Out.Contents()
			dstManifest := ORAS("manifest", "fetch", dst).Exec().Out.Contents()
			gomega.Expect(srcManifest).To(gomega.Equal(dstManifest))
		}

		It("should copy an image to a new repository via tag", func() {
			src := Reference(Host, Repo, FoobarImageTag)
			dst := Reference(Host, cpRepo("copy-tag"), "copiedTag")
			ORAS("cp", src, dst, "-v").MatchStatus(imageStateKeys, true, len(imageStateKeys)).Exec()
			validate(src, dst)
		})

		It("should copy an image to a new repository via tag without tagging", func() {
			src := Reference(Host, Repo, FoobarImageTag)
			dst := Reference(Host, cpRepo("copy-no-tagging"), FoobarImageDigest)
			ORAS("cp", src, dst, "-v").MatchStatus(imageStateKeys, true, len(imageStateKeys)).Exec()
			validate(src, dst)
		})

		It("should copy an image to a new repository via digest", func() {
			src := Reference(Host, Repo, FoobarImageDigest)
			dst := Reference(Host, cpRepo("copy-digest"), "copiedTag")
			ORAS("cp", src, dst, "-v").MatchStatus(imageStateKeys, true, len(imageStateKeys)).Exec()
			validate(src, dst)
		})

		It("should copy an image to a new repository with multiple tagging", func() {
			src := Reference(Host, Repo, FoobarImageDigest)
			tags := []string{"tag1", "tag2", "tag3"}
			dstRepo := cpRepo("copy-multi-tagging")
			dst := Reference(Host, dstRepo, "")
			ORAS("cp", src, dst+":"+strings.Join(tags, ","), "-v").MatchStatus(imageStateKeys, true, len(imageStateKeys)).Exec()
			for _, tag := range tags {
				dst := Reference(Host, dstRepo, tag)
				validate(src, dst)
			}
		})
	})
})
