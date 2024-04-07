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
	"regexp"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"oras.land/oras/test/e2e/internal/testdata/multi_arch"
	. "oras.land/oras/test/e2e/internal/utils"
)

var _ = Describe("ORAS beginners:", func() {
	When("running repo command", func() {
		It("should fail when no manifest reference provided", func() {
			ORAS("tag").ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})

		It("should fail when provided manifest reference is not found", func() {
			ORAS("tag", RegistryRef(ZOTHost, ImageRepo, InvalidTag), "tagged").ExpectFailure().MatchErrKeyWords(RegistryErrorPrefix).Exec()
		})

		It("should fail when the given reference is invalid", func() {
			ORAS("tag", fmt.Sprintf("%s/%s:%s", ZOTHost, InvalidRepo, "test"), "latest").ExpectFailure().MatchErrKeyWords("Error:", "unable to add tag", "invalid reference").Exec()
		})

		It("should fail and show detailed error description if no argument provided", func() {
			err := ORAS("tag").ExpectFailure().Exec().Err
			gomega.Expect(err).Should(gbytes.Say("Error"))
			gomega.Expect(err).Should(gbytes.Say("\nUsage: oras tag"))
			gomega.Expect(err).Should(gbytes.Say("\n"))
			gomega.Expect(err).Should(gbytes.Say(`Run "oras tag -h"`))
		})

		It("should fail with suggestion if calling with `tag list`", func() {
			ORAS("tag", "list").ExpectFailure().MatchErrKeyWords("Error:", `there is no "list" sub-command for "oras tag" command`, "repository", "oras repo tags").Exec()
			ORAS("tag", "list", Flags.Layout).ExpectFailure().MatchErrKeyWords("Error:", `there is no "list" sub-command for "oras tag" command`, "OCI Image Layout", "oras repo tags").Exec()
		})
	})
})

var _ = Describe("1.1 registry users:", func() {
	var tagAndValidate = func(reg string, repo string, tagOrDigest string, digest string, tags ...string) {
		out := ORAS(append([]string{"tag", RegistryRef(reg, repo, tagOrDigest)}, tags...)...).MatchKeyWords(tags...).Exec().Out
		hint := regexp.QuoteMeta(fmt.Sprintf("Tagging [registry] %s", RegistryRef(reg, repo, digest)))
		gomega.Expect(out).To(gbytes.Say(hint))
		gomega.Expect(out).NotTo(gbytes.Say(hint)) // should only say hint once
		ORAS("repo", "tags", RegistryRef(reg, repo, "")).MatchKeyWords(tags...).Exec()
	}
	When("running `tag`", func() {
		It("should add a tag to an existent manifest when providing tag reference", func() {
			tagAndValidate(ZOTHost, ImageRepo, multi_arch.Tag, multi_arch.Digest, "tag-via-tag")
		})
		It("should add a tag to an existent manifest when providing digest reference", func() {
			tagAndValidate(ZOTHost, ImageRepo, multi_arch.Digest, multi_arch.Digest, "tag-via-digest")
		})
		It("should add multiple tags to an existent manifest when providing digest reference", func() {
			tagAndValidate(ZOTHost, ImageRepo, multi_arch.Digest, multi_arch.Digest, "tag1-via-digest", "tag2-via-digest", "tag3-via-digest")
		})
		It("should add multiple tags to an existent manifest when providing tag reference", func() {
			tagAndValidate(ZOTHost, ImageRepo, multi_arch.Tag, multi_arch.Digest, "tag1-via-tag", "tag1-via-tag", "tag1-via-tag")
		})
	})
})

var _ = Describe("OCI image layout users:", func() {
	var tagAndValidate = func(root string, tagOrDigest string, digest string, tags ...string) {
		out := ORAS(append([]string{"tag", LayoutRef(root, tagOrDigest), Flags.Layout}, tags...)...).MatchKeyWords(tags...).Exec().Out
		hint := regexp.QuoteMeta(fmt.Sprintf("Tagging [oci-layout] %s", LayoutRef(root, digest)))
		gomega.Expect(out).To(gbytes.Say(hint))
		gomega.Expect(out).NotTo(gbytes.Say(hint)) // should only say hint once
		ORAS("repo", "tags", Flags.Layout, LayoutRef(root, "")).MatchKeyWords(tags...).Exec()
	}
	When("running `tag`", func() {
		It("should add a tag to an existent manifest when providing tag reference", func() {
			tagAndValidate(PrepareTempOCI(ImageRepo), multi_arch.Tag, multi_arch.Digest, "tag-via-tag")
		})
		It("should add a tag to an existent manifest when providing digest reference", func() {
			tagAndValidate(PrepareTempOCI(ImageRepo), multi_arch.Digest, multi_arch.Digest, "tag-via-digest")
		})
		It("should add multiple tags to an existent manifest when providing digest reference", func() {
			tagAndValidate(PrepareTempOCI(ImageRepo), multi_arch.Digest, multi_arch.Digest, "tag1-via-digest", "tag2-via-digest", "tag3-via-digest")
		})
		It("should add multiple tags to an existent manifest when providing tag reference", func() {
			tagAndValidate(PrepareTempOCI(ImageRepo), multi_arch.Tag, multi_arch.Digest, "tag1-via-tag", "tag1-via-tag", "tag1-via-tag")
		})
		It("should add a tag to an existent manifest when providing tag reference", func() {
			tagAndValidate(PrepareTempOCI(ImageRepo), multi_arch.Tag, multi_arch.Digest, "tag-via-tag")
		})
		It("should add a tag to an existent manifest when providing digest reference", func() {
			tagAndValidate(PrepareTempOCI(ImageRepo), multi_arch.Digest, multi_arch.Digest, "tag-via-digest")
		})
		It("should add multiple tags to an existent manifest when providing digest reference", func() {
			tagAndValidate(PrepareTempOCI(ImageRepo), multi_arch.Digest, multi_arch.Digest, "tag1-via-digest", "tag2-via-digest", "tag3-via-digest")
		})
		It("should add multiple tags to an existent manifest when providing tag reference", func() {
			tagAndValidate(PrepareTempOCI(ImageRepo), multi_arch.Tag, multi_arch.Digest, "tag1-via-tag", "tag1-via-tag", "tag1-via-tag")
		})
		It("should be able to retag a manifest at the current directory", func() {
			root := PrepareTempOCI(ImageRepo)
			dir := filepath.Dir(root)
			ref := filepath.Base(root)
			ORAS("tag", LayoutRef(ref, multi_arch.Tag), Flags.Layout, "latest").WithWorkDir(dir).MatchKeyWords("Tagging [oci-layout]", "Tagged latest").Exec()
			ORAS("tag", LayoutRef(ref, multi_arch.Tag), Flags.Layout, "tag2").WithWorkDir(dir).MatchKeyWords("Tagging [oci-layout]", "Tagged tag2").Exec()
			ORAS("repo", "tags", Flags.Layout, LayoutRef(ref, multi_arch.Tag)).WithWorkDir(dir).MatchKeyWords(multi_arch.Tag, "latest", "tag2").Exec()
		})
	})
})
