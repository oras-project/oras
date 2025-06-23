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

			It("should output tags in JSON format when using --format json flag", func() {
				// Use the existing repository
				repoRef := RegistryRef(ZOTHost, ImageRepo, "")

				// Run repo tags with JSON format
				bytes := ORAS("repo", "tags", repoRef, "--format", "json").Exec().Out.Contents()

				// Validate the JSON output
				var result struct {
					Tags []string `json:"tags"`
				}
				Expect(json.Unmarshal(bytes, &result)).ShouldNot(HaveOccurred())

				// Verify tags are in the output (using the known tags from the standard repository)
				Expect(result.Tags).Should(ContainElements(multi_arch.Tag, foobar.Tag))

				// Verify that the expected structure is present
				jsonString := string(bytes)
				Expect(jsonString).To(ContainSubstring(`"tags"`))
			})

			It("should handle digest exclusion with JSON format output", func() {
				// Prepare a repository with a digest-like tag
				repo := fmt.Sprintf("command/repo-tags-json-digest/%d", GinkgoRandomSeed())
				normalTag := "normal-tag"
				digestLikeTag := "sha256-e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

				// Create repository with tags
				ORAS("cp", RegistryRef(ZOTHost, ImageRepo, foobar.Tag), RegistryRef(ZOTHost, repo, normalTag)).
					WithDescription("prepare test repo with normal tag").Exec()
				ORAS("cp", RegistryRef(ZOTHost, ImageRepo, foobar.Tag), RegistryRef(ZOTHost, repo, digestLikeTag)).
					WithDescription("prepare test repo with digest-like tag").Exec()

				// Verify both tags are listed without exclusion
				plainBytes := ORAS("repo", "tags", RegistryRef(ZOTHost, repo, ""), "--format", "json").Exec().Out.Contents()
				var plainResult struct {
					Tags []string `json:"tags"`
				}
				Expect(json.Unmarshal(plainBytes, &plainResult)).ShouldNot(HaveOccurred())
				Expect(plainResult.Tags).Should(ContainElements(normalTag, digestLikeTag))

				// Run repo tags with JSON format and exclude digest tags
				bytes := ORAS("repo", "tags", RegistryRef(ZOTHost, repo, ""), "--format", "json", "--exclude-digest-tags").Exec().Out.Contents()

				// Validate the JSON output
				var result struct {
					Tags []string `json:"tags"`
				}
				Expect(json.Unmarshal(bytes, &result)).ShouldNot(HaveOccurred())

				// Verify only normal tag is in the output
				Expect(result.Tags).Should(ContainElement(normalTag))
				Expect(result.Tags).ShouldNot(ContainElement(digestLikeTag))
				Expect(result.Tags).Should(HaveLen(1))
			})

			It("should output tags in Go template format when using --format go-template flag", func() {
				// Use the existing repository
				repoRef := RegistryRef(ZOTHost, ImageRepo, "")

				// Run repo tags with Go template format
				template := "{{range .tags}}{{println .}}{{end}}"
				output := ORAS("repo", "tags", repoRef, "--format", "go-template="+template).Exec().Out.Contents()

				// Verify tags are in the output - should be one tag per line
				outputString := string(output)
				Expect(outputString).To(ContainSubstring(foobar.Tag))
				Expect(outputString).To(ContainSubstring(multi_arch.Tag))

				// Expected number of lines should match number of known tags
				expectedLines := 2 // foobar.Tag and multi_arch.Tag
				Expect(strings.Count(outputString, "\n")).To(Equal(expectedLines))
			})

			It("should handle empty repositories in JSON format", func() {
				// Create an empty repository with unique name
				emptyRepo := fmt.Sprintf("command/empty-repo-tags-json/%d", GinkgoRandomSeed())

				// Create the repository by pushing and then deleting content
				ref := RegistryRef(ZOTHost, emptyRepo, "temp-tag")
				tempDir := PrepareTempFiles()
				ORAS("push", ref, foobar.FileBarName).
					WithWorkDir(tempDir).
					WithDescription("create temporary repo").
					Exec()

				// Delete the manifest to make the repo empty
				ORAS("manifest", "delete", ref).
					WithDescription("delete manifest to make repo empty").
					Exec()

				// Verify the repository exists but has no tags in text mode
				noTagsSession := ORAS("repo", "tags", RegistryRef(ZOTHost, emptyRepo, "")).
					WithDescription("verify repo exists with no tags").
					Exec()
				Expect(noTagsSession.Out.Contents()).To(BeEmpty())

				// Run repo tags with JSON format on the empty repo
				bytes := ORAS("repo", "tags", RegistryRef(ZOTHost, emptyRepo, ""), "--format", "json").
					WithDescription("get JSON format of empty repo").
					Exec().Out.Contents()

				// Validate the JSON output
				var result struct {
					Tags []string `json:"tags"`
				}
				Expect(json.Unmarshal(bytes, &result)).ShouldNot(HaveOccurred())

				// Verify tags field exists and is an empty array
				Expect(result.Tags).Should(BeEmpty())

				// Verify the JSON structure is correct
				jsonString := string(bytes)
				Expect(jsonString).To(ContainSubstring(`"tags":[]`))
			})

			It("should return proper error format when using invalid repo with JSON format", func() {
				// Run repo tags with JSON format on a non-existent repository
				nonExistentRepo := fmt.Sprintf("non-existent-repo-%d", GinkgoRandomSeed())

				// Should fail the same way regardless of output format
				err := ORAS("repo", "tags", RegistryRef(ZOTHost, nonExistentRepo, ""), "--format", "json").
					ExpectFailure().
					MatchErrKeyWords(RegistryErrorPrefix).
					Exec().Err

				// Error should contain standard error message
				gomega.Expect(err).Should(gbytes.Say("Error"))

				// Compare with normal error output to ensure consistency
				regularErr := ORAS("repo", "tags", RegistryRef(ZOTHost, nonExistentRepo, "")).
					ExpectFailure().
					Exec().Err

				// Both error messages should indicate the same registry error
				gomega.Expect(regularErr).Should(gbytes.Say(RegistryErrorPrefix))
			})

			It("should output tags in OCI layout with JSON format", func() {
				// Use existing OCI layout tests for JSON output format
				root := PrepareTempOCI(ImageRepo)

				// Test JSON output for OCI layout
				bytes := ORAS("repo", "tags", root, "--format", "json", Flags.Layout).Exec().Out.Contents()

				// Parse the JSON output
				var result struct {
					Tags []string `json:"tags"`
				}
				Expect(json.Unmarshal(bytes, &result)).ShouldNot(HaveOccurred())

				// Verify expected tag is in output
				Expect(result.Tags).Should(ContainElement(foobar.Tag))

				// Verify JSON structure
				jsonString := string(bytes)
				Expect(jsonString).To(ContainSubstring(`"tags"`))
				Expect(jsonString).To(ContainSubstring(foobar.Tag))
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
		It("should show deprecation message when running with --verbose flag", func() {
			ORAS("repository", "list", ZOTHost, "--verbose").MatchErrKeyWords(feature.DeprecationMessageVerboseFlag).Exec()
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
		It("should show deprecation message when running with --verbose flag", func() {
			ORAS("repo", "tags", repoRef, "--verbose").MatchErrKeyWords(feature.DeprecationMessageVerboseFlag).Exec()
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
			viaTag := ORAS("repo", "tags", RegistryRef(ZOTHost, repo, foobar.Tag)).
				MatchKeyWords(tags...).
				MatchErrKeyWords(feature.Experimental.Mark, foobar.Digest).Exec().Out
			Expect(viaTag).ShouldNot(gbytes.Say(multi_arch.Tag))

			viaDigest := ORAS("repo", "tags", RegistryRef(ZOTHost, repo, foobar.Digest)).
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

		It("should list tags in JSON format for OCI layout", func() {
			// Use existing layout
			root := PrepareTempOCI(ImageRepo)

			// Add an additional tag to make more interesting test
			extra := "test-json-tag"
			ORAS("tag", LayoutRef(root, foobar.Tag), extra, Flags.Layout).
				WithDescription("prepare additional tag in OCI layout").
				Exec()

			// Run repo tags with JSON format
			bytes := ORAS("repository", "tags", root, "--format", "json", Flags.Layout).Exec().Out.Contents()

			// Parse the JSON output
			var result struct {
				Tags []string `json:"tags"`
			}
			Expect(json.Unmarshal(bytes, &result)).ShouldNot(HaveOccurred())

			// Verify both tags are included in the output
			Expect(result.Tags).Should(ContainElements(foobar.Tag, extra))
			Expect(result.Tags).Should(HaveLen(2))
		})

		It("should list out tags associated to the provided reference", func() {
			// prepare
			tags := []string{foobar.Tag, "bax", "bay", "baz"}
			root := prepare(ImageRepo, foobar.Tag, tags...)
			// test
			viaTag := ORAS("repo", "tags", LayoutRef(root, foobar.Tag), Flags.Layout).
				WithDescription("via tag").
				MatchKeyWords(tags...).
				MatchErrKeyWords(feature.Experimental.Mark, foobar.Digest).Exec().Out
			Expect(viaTag).ShouldNot(gbytes.Say(multi_arch.Tag))
			viaDigest := ORAS("repo", "tags", LayoutRef(root, foobar.Digest), Flags.Layout).
				WithDescription("via digest").
				MatchKeyWords(tags...).
				MatchErrKeyWords(feature.Experimental.Mark, foobar.Digest).Exec().Out
			Expect(viaDigest).ShouldNot(gbytes.Say(multi_arch.Tag))
		})
	})
})
