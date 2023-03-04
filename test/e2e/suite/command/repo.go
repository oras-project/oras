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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	"oras.land/oras/test/e2e/internal/testdata/multi_arch"
	. "oras.land/oras/test/e2e/internal/utils"
)

var _ = Describe("ORAS beginners:", func() {
	When("running repo command", func() {
		RunAndShowPreviewInHelp([]string{"repo"})
		When("running `repo ls`", func() {
			It("should show preview in help", func() {
				ORAS("repo", "ls", "--help").MatchKeyWords("[Preview] List", PreviewDesc, ExampleDesc).Exec()
			})

			It("should call sub-commands with aliases", func() {
				ORAS("repository", "list", "--help").MatchKeyWords("[Preview] List", PreviewDesc, ExampleDesc).Exec()
			})

			It("should fail listing repositories if wrong registry provided", func() {
				ORAS("repo", "ls").ExpectFailure().MatchErrKeyWords("Error:").Exec()
				ORAS("repo", "ls", Reference(Host, Repo, "some-tag")).ExpectFailure().MatchErrKeyWords("Error:").Exec()
			})
		})
		When("running `repo tags`", func() {
			It("should show preview in help", func() {
				ORAS("repo", "tags", "--help").MatchKeyWords("[Preview] Show tags", PreviewDesc, ExampleDesc).Exec()
			})

			It("should call sub-commands with aliases", func() {
				ORAS("repository", "show-tags", "--help").MatchKeyWords("[Preview] Show tags", PreviewDesc, ExampleDesc).Exec()
			})

			It("should fail listing repositories if wrong registry provided", func() {
				ORAS("repo", "tags").ExpectFailure().MatchErrKeyWords("Error:").Exec()
				ORAS("repo", "tags", Host).ExpectFailure().MatchErrKeyWords("Error:").Exec()
				ORAS("repo", "tags", Reference(Host, ImageRepo, "some-tag")).ExpectFailure().MatchErrKeyWords("Error:").Exec()
			})
		})
	})
})

var _ = Describe("Common registry users:", func() {
	When("running `repo ls`", func() {
		It("should list repositories", func() {
			ORAS("repository", "list", Host).MatchKeyWords(ImageRepo).Exec()
		})
		It("should list repositories under provided namespace", func() {
			ORAS("repo", "ls", Reference(Host, Namespace, "")).MatchKeyWords(Repo[len(Namespace)+1:]).Exec()
		})

		It("should not list repositories without a fully matched namespace", func() {
			repo := "command-draft/images"
			ORAS("cp", Reference(Host, Repo, foobar.Tag), Reference(Host, repo, foobar.Tag)).
				WithDescription("prepare destination repo: " + repo).
				Exec()
			ORAS("repo", "ls", Host).MatchKeyWords(Repo, repo).Exec()
			session := ORAS("repo", "ls", Reference(Host, Namespace, "")).MatchKeyWords(Repo[len(Namespace)+1:]).Exec()
			Expect(session.Out).ShouldNot(gbytes.Say(repo[len(Namespace)+1:]))
		})

		It("should list repositories via short command", func() {
			ORAS("repo", "ls", Host).MatchKeyWords(ImageRepo).Exec()
		})
		It("should list partial repositories via `last` flag", func() {
			session := ORAS("repo", "ls", Host, "--last", ImageRepo).Exec()
			repoRegex := regexp.QuoteMeta(ImageRepo + "\n")
			Expect(session.Out).ShouldNot(gbytes.Say(repoRegex))
		})

	})
	When("running `repo tags`", func() {
		repoWithName := func(name string) string {
			return fmt.Sprintf("command/images/repo/tags/%d/%s", GinkgoRandomSeed(), name)
		}
		repoRef := Reference(Host, ImageRepo, "")
		It("should list tags", func() {
			ORAS("repository", "show-tags", repoRef).MatchKeyWords(multi_arch.Tag, foobar.Tag).Exec()
		})
		It("should list tags via short command", func() {
			ORAS("repo", "tags", repoRef).MatchKeyWords(multi_arch.Tag, foobar.Tag).Exec()

		})
		It("should list partial tags via `last` flag", func() {
			session := ORAS("repo", "tags", repoRef, "--last", foobar.Tag).MatchKeyWords(multi_arch.Tag).Exec()
			Expect(session.Out).ShouldNot(gbytes.Say(foobar.Tag))
		})

		It("Should list out tags associated to the provided reference", func() {
			// prepare
			repo := repoWithName("filter-tag")
			tags := []string{foobar.Tag, "bax", "bay", "baz"}
			refWithTags := fmt.Sprintf("%s:%s", Reference(Host, repo, ""), strings.Join(tags, ","))
			ORAS("cp", Reference(Host, Repo, foobar.Tag), refWithTags).
				WithDescription("prepare: copy and create multiple tags to " + refWithTags).
				Exec()
			ORAS("cp", Reference(Host, Repo, multi_arch.Tag), Reference(Host, Repo, "")).
				WithDescription("prepare: copy tag with different digest").
				Exec()
			// test
			viaTag := ORAS("repo", "tags", "-v", Reference(Host, repo, foobar.Tag)).
				MatchKeyWords(tags...).
				MatchErrKeyWords("Preview", foobar.Digest).Exec().Out
			Expect(viaTag).ShouldNot(gbytes.Say(multi_arch.Tag))

			viaDigest := ORAS("repo", "tags", "-v", Reference(Host, repo, foobar.Digest)).
				MatchKeyWords(tags...).
				MatchErrKeyWords("Preview", foobar.Digest).Exec().Out
			Expect(viaDigest).ShouldNot(gbytes.Say(multi_arch.Tag))
		})
	})
})
