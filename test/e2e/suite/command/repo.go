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
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
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
				ORAS("repo", "ls").WithFailureCheck().MatchErrKeyWords("Error:").Exec()
				ORAS("repo", "ls", Reference(Host, Repo, "")).WithFailureCheck().MatchErrKeyWords("Error:").Exec()
				ORAS("repo", "ls", Reference(Host, Repo, "some-tag")).WithFailureCheck().MatchErrKeyWords("Error:").Exec()
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
				ORAS("repo", "tags").WithFailureCheck().MatchErrKeyWords("Error:").Exec()
				ORAS("repo", "tags", Host).WithFailureCheck().MatchErrKeyWords("Error:").Exec()
				ORAS("repo", "tags", Reference(Host, Repo, "some-tag")).WithFailureCheck().MatchErrKeyWords("Error:").Exec()
			})
		})
	})
})

var _ = Describe("Common registry users:", func() {
	When("running `repo ls`", func() {
		It("should list repositories", func() {
			ORAS("repository", "list", Host).MatchKeyWords(Repo).Exec()
		})
		It("should list repositories via short command", func() {
			ORAS("repo", "ls", Host).MatchKeyWords(Repo).Exec()
		})
		It("should list partial repositories via `last` flag", func() {
			session := ORAS("repo", "ls", Host, "--last", Repo).Exec()
			Expect(session.Out).ShouldNot(gbytes.Say(Repo))
		})
	})
	When("running `repo tags`", func() {
		repoRef := Reference(Host, Repo, "")
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
	})
})
