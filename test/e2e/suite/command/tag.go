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
	"regexp"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"oras.land/oras/test/e2e/internal/testdata/multi_arch"
	. "oras.land/oras/test/e2e/internal/utils"
)

var _ = Describe("ORAS beginners:", func() {
	When("running repo command", func() {
		RunAndShowPreviewInHelp([]string{"tag"})
		It("should fail when no manifest reference provided", func() {
			ORAS("tag").ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})

		It("should fail when provided manifest reference is not found", func() {
			ORAS("tag", Reference(Host, ImageRepo, "i-dont-think-this-tag-exists")).ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})
	})
})

var _ = Describe("Common registry users:", func() {
	var tagAndValidate = func(reg string, repo string, tagOrDigest string, digest string, tags ...string) {
		out := ORAS(append([]string{"tag", Reference(reg, repo, tagOrDigest)}, tags...)...).MatchKeyWords(tags...).Exec().Out
		hint := regexp.QuoteMeta(fmt.Sprintf("Tagging [registry] %s", Reference(reg, repo, digest)))
		gomega.Expect(out).To(gbytes.Say(hint))
		gomega.Expect(out).NotTo(gbytes.Say(hint)) // should only say hint once
		ORAS("repo", "tags", Reference(reg, repo, "")).MatchKeyWords(tags...).Exec()
	}
	When("running `tag`", func() {
		It("should add a tag to an existent manifest when providing tag reference", func() {
			tagAndValidate(Host, ImageRepo, multi_arch.Tag, multi_arch.Digest, "tag-via-tag")
		})
		It("should add a tag to an existent manifest when providing digest reference", func() {
			tagAndValidate(Host, ImageRepo, multi_arch.Digest, multi_arch.Digest, "tag-via-digest")
		})
		It("should add multiple tags to an existent manifest when providing digest reference", func() {
			tagAndValidate(Host, ImageRepo, multi_arch.Digest, multi_arch.Digest, "tag1-via-digest", "tag2-via-digest", "tag3-via-digest")
		})
		It("should add multiple tags to an existent manifest when providing tag reference", func() {
			tagAndValidate(Host, ImageRepo, multi_arch.Tag, multi_arch.Digest, "tag1-via-tag", "tag1-via-tag", "tag1-via-tag")
		})
	})
})
