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
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
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
			ORAS("cp", Reference(Host, Repo, FoobarImageTag), Reference(Host, repo, FoobarImageTag)).
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
			Expect(session.Out).ShouldNot(gbytes.Say(ImageRepo))
		})

	})
	When("running `repo tags`", func() {
		repoWithName := func(name string) string {
			return fmt.Sprintf("command/images/repo/tags/%d/%s", GinkgoRandomSeed(), name)
		}
		repoRef := Reference(Host, ImageRepo, "")
		It("should list tags", func() {
			ORAS("repository", "show-tags", repoRef).MatchKeyWords(MultiImageTag, FoobarImageTag).Exec()
		})
		It("should list tags via short command", func() {
			ORAS("repo", "tags", repoRef).MatchKeyWords(MultiImageTag, FoobarImageTag).Exec()

		})
		It("should list partial tags via `last` flag", func() {
			session := ORAS("repo", "tags", repoRef, "--last", FoobarImageTag).MatchKeyWords(MultiImageTag).Exec()
			Expect(session.Out).ShouldNot(gbytes.Say(FoobarImageTag))
		})

		It("Should list out tags associated to the provided reference", func() {
			// prepare
			repo := repoWithName("filter-tag")
			tags := []string{foobar.Tag, "foo", "bar", "baz"}
			refWithTags := fmt.Sprintf("%s:%s", Reference(Host, repo, ""), strings.Join(tags, ","))
			ORAS("cp", Reference(Host, Repo, foobar.Tag), refWithTags).
				WithDescription("prepare: copy and create multiple tags to " + refWithTags).
				Exec()
			ORAS("cp", Reference(Host, Repo, MultiImageTag), refWithTags).
				WithDescription("prepare: copy tag with different digest").
				Exec()
			// test
			viaTag := ORAS("repo", "tags", Reference(Host, repo, foobar.Tag)).
				MatchKeyWords(tags...).
				MatchErrKeyWords("Preview").Exec().Out
			Expect(viaTag).ShouldNot(gbytes.Say(MultiImageTag))

			viaDigest := ORAS("repo", "tags", Reference(Host, repo, foobar.Digest)).
				MatchKeyWords(tags...).
				MatchErrKeyWords("Preview").Exec().Out
			Expect(viaDigest).ShouldNot(gbytes.Say(MultiImageTag))
		})
	})
})
