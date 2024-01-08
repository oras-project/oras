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
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"oras.land/oras/test/e2e/internal/testdata/feature"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	"oras.land/oras/test/e2e/internal/testdata/multi_arch"
	. "oras.land/oras/test/e2e/internal/utils"
)

var _ = Describe("ORAS beginners:", func() {
	When("running repo command", func() {
		When("running `repo ls`", func() {
			It("should show help description", func() {
				ORAS("repo", "ls", "--help").MatchKeyWords(ExampleDesc).Exec()
			})

			It("should call sub-commands with aliases", func() {
				ORAS("repository", "list", "--help").MatchKeyWords(ExampleDesc).Exec()
			})

			It("should fail listing repositories if wrong registry provided", func() {
				ORAS("repo", "ls", RegistryRef(ZOTHost, ImageRepo, "some-tag")).ExpectFailure().MatchErrKeyWords("Error:").Exec()
			})

			It("should fail and show detailed error description if no argument provided", func() {
				err := ORAS("repo", "ls").ExpectFailure().Exec().Err
				gomega.Expect(err).Should(gbytes.Say("Error"))
				gomega.Expect(err).Should(gbytes.Say("\nUsage: oras repo ls"))
				gomega.Expect(err).Should(gbytes.Say("\n"))
				gomega.Expect(err).Should(gbytes.Say(`Run "oras repo ls -h"`))
			})

			It("should fail if password is wrong with registry error prefix", func() {
				ORAS("repo", "ls", ZOTHost, "-u", Username, "-p", "???").
					MatchErrKeyWords(RegistryErrorPrefix).ExpectFailure().Exec()
			})
		})
		When("running `repo tags`", func() {
			It("should show help description with feature flags", func() {
				out := ORAS("repo", "tags", "--help").MatchKeyWords(ExampleDesc).Exec().Out
				Expect(out).Should(gbytes.Say("--exclude-digest-tags\\s+%s", regexp.QuoteMeta(feature.Preview.Mark)))
			})

			It("should call sub-commands with aliases", func() {
				ORAS("repository", "show-tags", "--help").MatchKeyWords(ExampleDesc).Exec()
			})

			It("should fail listing repositories if wrong reference provided", func() {
				ORAS("repo", "tags").ExpectFailure().MatchErrKeyWords("Error:").Exec()
				ORAS("repo", "tags", ZOTHost).ExpectFailure().MatchErrKeyWords("Error:").Exec()
				ORAS("repo", "tags", RegistryRef(ZOTHost, ImageRepo, "some-tag")).ExpectFailure().MatchErrKeyWords(RegistryErrorPrefix).Exec()
			})

			It("should fail and show detailed error description if no argument provided", func() {
				err := ORAS("repo", "tags").ExpectFailure().Exec().Err
				gomega.Expect(err).Should(gbytes.Say("Error"))
				gomega.Expect(err).Should(gbytes.Say("\nUsage: oras repo tags"))
				gomega.Expect(err).Should(gbytes.Say("\n"))
				gomega.Expect(err).Should(gbytes.Say(`Run "oras repo tags -h"`))
			})
		})
	})
})

var _ = Describe("1.1 registry users:", func() {
	When("running `repo ls`", func() {
		It("should list repositories", func() {
			ORAS("repository", "list", ZOTHost).MatchKeyWords(ImageRepo).Exec()
		})
		It("should list repositories under provided namespace", func() {
			ORAS("repo", "ls", RegistryRef(ZOTHost, Namespace, "")).MatchKeyWords(ImageRepo[len(Namespace)+1:]).Exec()
		})

		It("should not list repositories without a fully matched namespace", func() {
			repo := "command-draft/images"
			ORAS("cp", RegistryRef(ZOTHost, ImageRepo, foobar.Tag), RegistryRef(ZOTHost, repo, foobar.Tag)).
				WithDescription("prepare destination repo: " + repo).
				Exec()
			ORAS("repo", "ls", ZOTHost).MatchKeyWords(ImageRepo, repo).Exec()
			session := ORAS("repo", "ls", RegistryRef(ZOTHost, Namespace, "")).MatchKeyWords(ImageRepo[len(Namespace)+1:]).Exec()
			Expect(session.Out).ShouldNot(gbytes.Say(repo[len(Namespace)+1:]))
		})

		It("should list repositories via short command", func() {
			ORAS("repo", "ls", ZOTHost).MatchKeyWords(ImageRepo).Exec()
		})

	})
	When("running `repo tags`", func() {
		repoWithName := func(name string) string {
			return fmt.Sprintf("command/images/repo/tags/%d/%s", GinkgoRandomSeed(), name)
		}
		repoRef := RegistryRef(ZOTHost, ImageRepo, "")
		It("should list tags", func() {
			ORAS("repository", "show-tags", repoRef).MatchKeyWords(multi_arch.Tag, foobar.Tag).Exec()
		})
		It("should list tags via short command", func() {
			ORAS("repo", "tags", repoRef).MatchKeyWords(multi_arch.Tag, foobar.Tag).Exec()

		})

		It("Should list out tags associated to the provided reference", func() {
			// prepare
			repo := repoWithName("filter-tag")
			tags := []string{foobar.Tag, "bax", "bay", "baz"}
			refWithTags := fmt.Sprintf("%s:%s", RegistryRef(ZOTHost, repo, ""), strings.Join(tags, ","))
			ORAS("cp", RegistryRef(ZOTHost, ImageRepo, foobar.Tag), refWithTags).
				WithDescription("prepare: copy and create multiple tags to " + refWithTags).
				Exec()
			ORAS("cp", RegistryRef(ZOTHost, ImageRepo, multi_arch.Tag), RegistryRef(ZOTHost, ImageRepo, "")).
				WithDescription("prepare: copy tag with different digest").
				Exec()
			// test
			viaTag := ORAS("repo", "tags", "-v", RegistryRef(ZOTHost, repo, foobar.Tag)).
				MatchKeyWords(tags...).
				MatchErrKeyWords(feature.Experimental.Mark, foobar.Digest).Exec().Out
			Expect(viaTag).ShouldNot(gbytes.Say(multi_arch.Tag))

			viaDigest := ORAS("repo", "tags", "-v", RegistryRef(ZOTHost, repo, foobar.Digest)).
				MatchKeyWords(tags...).
				MatchErrKeyWords(feature.Experimental.Mark, foobar.Digest).Exec().Out
			Expect(viaDigest).ShouldNot(gbytes.Say(multi_arch.Tag))
		})
	})
})

var _ = Describe("1.0 registry users:", func() {
	When("running `repo ls`", func() {
		It("should list partial repositories via `last` flag", func() {
			session := ORAS("repo", "ls", FallbackHost, "--last", ArtifactRepo).Exec()
			repoRegex := regexp.QuoteMeta(ImageRepo + "\n")
			Expect(session.Out).ShouldNot(gbytes.Say(repoRegex))
		})
	})
})

var _ = Describe("OCI image layout users:", func() {
	When("running `repo tags`", func() {
		prepare := func(repo string, fromTag string, toTags ...string) string {
			root := PrepareTempOCI(repo)
			ORAS("tag", LayoutRef(root, fromTag), strings.Join(toTags, " "), Flags.Layout).
				WithDescription("prepare in OCI layout").
				Exec()
			return root
		}
		tagOutput := foobar.Tag + "\n"
		It("should list tags", func() {
			root := PrepareTempOCI(ImageRepo)
			ORAS("repository", "show-tags", root, Flags.Layout).MatchKeyWords(tagOutput).Exec()
		})
		It("should list tags via short command", func() {
			root := PrepareTempOCI(ImageRepo)
			ORAS("repository", "tags", root, Flags.Layout).MatchKeyWords(tagOutput).Exec()
		})
		It("should list partial tags via `last` flag", func() {
			// prepare
			extra := "zzz"
			root := prepare(ImageRepo, foobar.Tag, extra)
			// test
			session := ORAS("repository", "tags", root, "--last", foobar.Tag, Flags.Layout).MatchKeyWords(extra).Exec()
			Expect(session.Out).ShouldNot(gbytes.Say(regexp.QuoteMeta(tagOutput)))
		})

		It("should list out tags associated to the provided reference", func() {
			// prepare
			tags := []string{foobar.Tag, "bax", "bay", "baz"}
			root := prepare(ImageRepo, foobar.Tag, tags...)
			// test
			viaTag := ORAS("repo", "tags", "-v", LayoutRef(root, foobar.Tag), Flags.Layout).
				WithDescription("via tag").
				MatchKeyWords(tags...).
				MatchErrKeyWords(feature.Experimental.Mark, foobar.Digest).Exec().Out
			Expect(viaTag).ShouldNot(gbytes.Say(multi_arch.Tag))
			viaDigest := ORAS("repo", "tags", "-v", LayoutRef(root, foobar.Digest), Flags.Layout).
				WithDescription("via digest").
				MatchKeyWords(tags...).
				MatchErrKeyWords(feature.Experimental.Mark, foobar.Digest).Exec().Out
			Expect(viaDigest).ShouldNot(gbytes.Say(multi_arch.Tag))
		})
	})
})
