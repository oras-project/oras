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
	. "github.com/onsi/ginkgo/v2"
	. "oras.land/oras/test/e2e/internal/utils"
)

var _ = Describe("ORAS beginners:", func() {
	When("running repo command", func() {
		RunAndShowPreviewInHelp([]string{"tag"})
		It("should fail when no manifest reference provided", func() {
			ORAS("tag").WithFailureCheck().MatchErrKeyWords("Error:").Exec()
		})

		It("should fail when provided manifest reference is not found", func() {
			ORAS("tag", Reference(Host, Repo, "i-dont-think-this-tag-exists")).WithFailureCheck().MatchErrKeyWords("Error:").Exec()
		})
	})
})

var _ = Describe("Common registry users:", func() {
	var tagAndValidate = func(reg string, repo string, tagOrDigest string, tags ...string) {
		ORAS(append([]string{"tag", Reference(reg, repo, tagOrDigest)}, tags...)...).MatchKeyWords(tags...).Exec()
		ORAS("repo", "tags", Reference(reg, repo, "")).MatchKeyWords(tags...).Exec()
	}
	When("running `tag`", func() {
		It("should add a tag to an existent manifest when providing tag reference", func() {
			tagAndValidate(Host, Repo, MultiImageTag, "tag-via-tag")
		})
		It("should add a tag to an existent manifest when providing digest reference", func() {
			tagAndValidate(Host, Repo, MultiImageDigest, "tag-via-digest")
		})
		It("should add multiple tags to an existent manifest when providing digest reference", func() {
			tagAndValidate(Host, Repo, MultiImageDigest, "tag1-via-digest", "tag2-via-digest", "tag3-via-digest")
		})
		It("should add multiple tags to an existent manifest when providing tag reference", func() {
			tagAndValidate(Host, Repo, MultiImageTag, "tag1-via-tag", "tag1-via-tag", "tag1-via-tag")
		})
	})
})
