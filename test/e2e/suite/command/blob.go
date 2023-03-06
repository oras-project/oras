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
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	. "oras.land/oras/test/e2e/internal/utils"
)

const (
	pushContent      = "test-blob"
	pushDigest       = "sha256:e1ca41574914ba00e8ed5c8fc78ec8efdfd48941c7e48ad74dad8ada7f2066d8"
	invalidDigest    = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	pushDescFmt      = `{"mediaType":"%s","digest":"sha256:e1ca41574914ba00e8ed5c8fc78ec8efdfd48941c7e48ad74dad8ada7f2066d8","size":9}`
	deleteDigest     = "sha256:fcde2b2edba56bf408601fb721fe9b5c338d10ee429ea04fae5511b68fbf8fb9"
	deleteDescriptor = `{"mediaType":"application/octet-stream","digest":"sha256:fcde2b2edba56bf408601fb721fe9b5c338d10ee429ea04fae5511b68fbf8fb9","size":3}`
	deleteContent    = "bar"
)

var _ = Describe("ORAS beginners:", func() {
	repoFmt := fmt.Sprintf("command/blob/%%s/%d/%%s", GinkgoRandomSeed())
	When("running blob command", func() {
		RunAndShowPreviewInHelp([]string{"blob"})

		When("running `blob push`", func() {
			RunAndShowPreviewInHelp([]string{"blob", "push"}, PreviewDesc, ExampleDesc)
			It("should fail to read blob content and password from stdin at the same time", func() {
				repo := fmt.Sprintf(repoFmt, "push", "password-stdin")
				ORAS("blob", "push", Reference(Host, repo, ""), "--password-stdin", "-").
					ExpectFailure().
					MatchTrimmedContent("Error: `-` read file from input and `--password-stdin` read password from input cannot be both used").Exec()
			})
			It("should fail to push a blob from stdin but no blob size provided", func() {
				repo := fmt.Sprintf(repoFmt, "push", "no-size")
				ORAS("blob", "push", Reference(Host, repo, pushDigest), "-").
					WithInput(strings.NewReader(pushContent)).
					ExpectFailure().
					MatchTrimmedContent("Error: `--size` must be provided if the blob is read from stdin").Exec()
			})

			It("should fail to push a blob from stdin if invalid blob size provided", func() {
				repo := fmt.Sprintf(repoFmt, "push", "invalid-stdin-size")
				ORAS("blob", "push", Reference(Host, repo, pushDigest), "-", "--size", "3").
					WithInput(strings.NewReader(pushContent)).ExpectFailure().
					Exec()
			})

			It("should fail to push a blob from stdin if invalid digest provided", func() {
				repo := fmt.Sprintf(repoFmt, "push", "invalid-stdin-digest")
				ORAS("blob", "push", Reference(Host, repo, invalidDigest), "-", "--size", strconv.Itoa(len(pushContent))).
					WithInput(strings.NewReader(pushContent)).ExpectFailure().
					Exec()
			})

			It("should fail to push a blob from file if invalid blob size provided", func() {
				repo := fmt.Sprintf(repoFmt, "push", "invalid-file-digest")
				blobPath := WriteTempFile("blob", pushContent)
				ORAS("blob", "push", Reference(Host, repo, pushDigest), blobPath, "--size", "3").
					ExpectFailure().
					Exec()
			})

			It("should fail to push a blob from file if invalid digest provided", func() {
				repo := fmt.Sprintf(repoFmt, "push", "invalid-stdin-size")
				blobPath := WriteTempFile("blob", pushContent)
				ORAS("blob", "push", Reference(Host, repo, invalidDigest), blobPath, "--size", strconv.Itoa(len(pushContent))).
					WithInput(strings.NewReader(pushContent)).ExpectFailure().
					Exec()
			})

			It("should fail if no reference is provided", func() {
				ORAS("blob", "push").ExpectFailure().Exec()
			})
		})

		When("running `blob fetch`", func() {
			RunAndShowPreviewInHelp([]string{"blob", "fetch"}, PreviewDesc, ExampleDesc)

			It("should call sub-commands with aliases", func() {
				ORAS("blob", "get", "--help").
					MatchKeyWords("[Preview] Fetch", PreviewDesc, ExampleDesc).
					Exec()
			})
			It("should have flag for prettifying JSON output", func() {
				ORAS("blob", "get", "--help").
					MatchKeyWords("--pretty", "prettify JSON").
					Exec()
			})

			It("should fail if neither output path nor descriptor flag are not provided", func() {
				ORAS("blob", "fetch", Reference(Host, ImageRepo, "sha256:2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae")).
					ExpectFailure().Exec()
			})

			It("should fail if no digest provided", func() {
				ORAS("blob", "fetch", Reference(Host, ImageRepo, "")).
					ExpectFailure().Exec()
			})

			It("should fail if provided digest doesn't existed", func() {
				ORAS("blob", "fetch", Reference(Host, ImageRepo, "sha256:2aaa2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a")).
					ExpectFailure().Exec()
			})

			It("should fail if output path points to stdout and descriptor flag is provided", func() {
				ORAS("blob", "fetch", Reference(Host, ImageRepo, ""), "--descriptor", "--output", "-").
					ExpectFailure().Exec()
			})

			It("should fail if no reference is provided", func() {
				ORAS("blob", "fetch").ExpectFailure().Exec()
			})
		})
	})

	When("running `blob delete`", func() {
		RunAndShowPreviewInHelp([]string{"blob", "delete"}, PreviewDesc, ExampleDesc)

		It("should fail if no blob reference is provided", func() {
			dstRepo := fmt.Sprintf(repoFmt, "delete", "no-ref")
			ORAS("cp", Reference(Host, ImageRepo, foobar.Digest), Reference(Host, dstRepo, foobar.Digest)).Exec()
			ORAS("blob", "delete").ExpectFailure().Exec()
			ORAS("blob", "fetch", Reference(Host, dstRepo, deleteDigest), "--output", "-").MatchContent(deleteContent).Exec()
		})

		It("should fail if no force flag and descriptor flag is provided", func() {
			dstRepo := fmt.Sprintf(repoFmt, "delete", "no-confirm")
			ORAS("cp", Reference(Host, ImageRepo, foobar.Digest), Reference(Host, dstRepo, foobar.Digest)).Exec()
			ORAS("blob", "delete", Reference(Host, dstRepo, deleteDigest), "--descriptor").ExpectFailure().Exec()
			ORAS("blob", "fetch", Reference(Host, dstRepo, deleteDigest), "--output", "-").MatchContent(deleteContent).Exec()
		})

		It("should fail if the blob reference is not in the form of <name@digest>", func() {
			dstRepo := fmt.Sprintf(repoFmt, "delete", "wrong-ref-form")
			ORAS("blob", "delete", fmt.Sprintf("%s/%s:%s", Host, dstRepo, "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), "--descriptor", "--force").ExpectFailure().Exec()
			ORAS("blob", "delete", fmt.Sprintf("%s/%s:%s", Host, dstRepo, "test"), "--descriptor", "--force").ExpectFailure().Exec()
			ORAS("blob", "delete", fmt.Sprintf("%s/%s@%s", Host, dstRepo, "test"), "--descriptor", "--force").ExpectFailure().Exec()
		})

		It("should fail to delete a non-existent blob without force flag set", func() {
			toDeleteRef := Reference(Host, ImageRepo, invalidDigest)
			ORAS("blob", "delete", toDeleteRef).
				ExpectFailure().
				MatchErrKeyWords(toDeleteRef, "the specified blob does not exist").
				Exec()
		})

		It("should fail to delete a non-existent blob and output descriptor, with force flag set", func() {
			toDeleteRef := Reference(Host, ImageRepo, invalidDigest)
			ORAS("blob", "delete", toDeleteRef, "--force", "--descriptor").
				ExpectFailure().
				MatchErrKeyWords(toDeleteRef, "the specified blob does not exist").
				Exec()
		})
	})
})

var _ = Describe("Common registry users:", func() {
	repoFmt := fmt.Sprintf("command/blob/%%s/%d/%%s", GinkgoRandomSeed())
	When("running `blob delete`", func() {
		It("should delete a blob with interactive confirmation", func() {
			dstRepo := fmt.Sprintf(repoFmt, "delete", "prompt-confirmation")
			ORAS("cp", Reference(Host, ImageRepo, foobar.Digest), Reference(Host, dstRepo, foobar.Digest)).Exec()
			toDeleteRef := Reference(Host, dstRepo, deleteDigest)
			ORAS("blob", "delete", toDeleteRef).
				WithInput(strings.NewReader("y")).
				MatchKeyWords("Deleted", toDeleteRef).Exec()
			ORAS("blob", "delete", toDeleteRef).
				WithDescription("validate").
				WithInput(strings.NewReader("y")).
				ExpectFailure().
				MatchErrKeyWords("Error:", toDeleteRef, "the specified blob does not exist").Exec()
		})

		It("should delete a blob with force flag and output descriptor", func() {
			dstRepo := fmt.Sprintf(repoFmt, "delete", "flag-confirmation")
			ORAS("cp", Reference(Host, ImageRepo, foobar.Digest), Reference(Host, dstRepo, foobar.Digest)).Exec()
			toDeleteRef := Reference(Host, dstRepo, deleteDigest)
			ORAS("blob", "delete", toDeleteRef, "--force", "--descriptor").MatchContent(deleteDescriptor).Exec()
			ORAS("blob", "delete", toDeleteRef).WithDescription("validate").ExpectFailure().MatchErrKeyWords("Error:", toDeleteRef, "the specified blob does not exist").Exec()
		})

		It("should return success when deleting a non-existent blob with force flag set", func() {
			toDeleteRef := Reference(Host, ImageRepo, invalidDigest)
			ORAS("blob", "delete", toDeleteRef, "--force").
				MatchKeyWords("Missing", toDeleteRef).
				Exec()
		})
	})
	When("running `blob push`", func() {
		It("should push a blob from a file and output the descriptor with specific media-type", func() {
			mediaType := "test.media"
			repo := fmt.Sprintf(repoFmt, "push", "blob-file-media-type")
			blobPath := WriteTempFile("blob", pushContent)
			ORAS("blob", "push", Reference(Host, repo, ""), blobPath, "--media-type", mediaType, "--descriptor").
				MatchContent(fmt.Sprintf(pushDescFmt, mediaType)).Exec()
			ORAS("blob", "fetch", Reference(Host, repo, pushDigest), "--output", "-").MatchContent(pushContent).Exec()

			ORAS("blob", "push", Reference(Host, repo, ""), blobPath, "-v").
				WithDescription("skip the pushing if the blob already exists in the target repo").
				MatchKeyWords("Exists").Exec()

		})

		It("should push a blob from a stdin and output the descriptor with specific media-type", func() {
			mediaType := "test.media"
			repo := fmt.Sprintf(repoFmt, "push", "blob-file-media-type")
			ORAS("blob", "push", Reference(Host, repo, pushDigest), "-", "--media-type", mediaType, "--descriptor", "--size", strconv.Itoa(len(pushContent))).
				WithInput(strings.NewReader(pushContent)).
				MatchContent(fmt.Sprintf(pushDescFmt, mediaType)).Exec()
			ORAS("blob", "fetch", Reference(Host, repo, pushDigest), "--output", "-").MatchContent(pushContent).Exec()
		})
	})

	When("running `blob fetch`", func() {
		It("should fetch blob descriptor ", func() {
			ORAS("blob", "fetch", Reference(Host, ImageRepo, foobar.Foo1BlobDigest), "--descriptor").
				MatchContent(foobar.Foo1BlobDescriptor).Exec()
		})
		It("should fetch blob content and output to stdout", func() {
			ORAS("blob", "fetch", Reference(Host, ImageRepo, foobar.Foo1BlobDigest), "--output", "-").
				MatchContent(foobar.Foo1BlobContent).Exec()
		})
		It("should fetch blob content and output to a file", func() {
			tempDir := GinkgoT().TempDir()
			contentPath := filepath.Join(tempDir, "fetched")
			ORAS("blob", "fetch", Reference(Host, ImageRepo, foobar.Foo1BlobDigest), "--output", contentPath).
				WithWorkDir(tempDir).Exec()
			MatchFile(contentPath, foobar.Foo1BlobContent, DefaultTimeout)
		})
		It("should fetch blob descriptor and output content to a file", func() {
			tempDir := GinkgoT().TempDir()
			contentPath := filepath.Join(tempDir, "fetched")
			ORAS("blob", "fetch", Reference(Host, ImageRepo, foobar.Foo1BlobDigest), "--output", contentPath, "--descriptor").
				MatchContent(foobar.Foo1BlobDescriptor).
				WithWorkDir(tempDir).Exec()
			MatchFile(contentPath, foobar.Foo1BlobContent, DefaultTimeout)
		})
	})
})

var _ = Describe("OCI image layout users:", Focus, func() {
	When("running `blob delete`", func() {
		It("should not support deleting a blob", func() {
			toDeleteRef := LayoutRef(PrepareTempLayout(), deleteDigest)
			ORAS("blob", "delete", LayoutFlag, toDeleteRef).
				WithInput(strings.NewReader("y")).
				MatchErrKeyWords("Error:", "unknown flag", LayoutFlag).
				ExpectFailure().
				Exec()
		})
	})

	When("running `blob fetch`", func() {
		root := filepath.Join(TestDataRoot, "image_layout", "oci")
		It("should fetch blob descriptor ", func() {
			ORAS("blob", "fetch", LayoutFlag, LayoutRef(root, foobar.Foo1BlobDigest), "--descriptor").
				MatchContent(foobar.Foo1BlobDescriptor).Exec()
		})
		It("should fetch blob content and output to stdout", func() {
			ORAS("blob", "fetch", LayoutFlag, LayoutRef(root, foobar.Foo1BlobDigest), "--output", "-").
				MatchContent(foobar.Foo1BlobContent).Exec()
		})
		It("should fetch blob content and output to a file", func() {
			tempDir := GinkgoT().TempDir()
			contentPath := filepath.Join(tempDir, "fetched")
			ORAS("blob", "fetch", LayoutFlag, LayoutRef(root, foobar.Foo1BlobDigest), "--output", contentPath).
				WithWorkDir(tempDir).Exec()
			MatchFile(contentPath, foobar.Foo1BlobContent, DefaultTimeout)
		})
		It("should fetch blob descriptor and output content to a file", func() {
			tempDir := GinkgoT().TempDir()
			contentPath := filepath.Join(tempDir, "fetched")
			ORAS("blob", "fetch", LayoutFlag, LayoutRef(root, foobar.Foo1BlobDigest), "--output", contentPath, "--descriptor").
				MatchContent(foobar.Foo1BlobDescriptor).
				WithWorkDir(tempDir).Exec()
			MatchFile(contentPath, foobar.Foo1BlobContent, DefaultTimeout)
		})
	})

	When("running `blob push`", func() {
		It("should push a blob from a file and output the descriptor with specific media-type", func() {
			// prepare
			tmpRoot := GinkgoT().TempDir()
			mediaType := "test.media"
			blobPath := WriteTempFile("blob", pushContent)
			// test
			ORAS("blob", "push", LayoutFlag, LayoutRef(tmpRoot, pushDigest), blobPath, "--media-type", mediaType, "--descriptor").
				MatchContent(fmt.Sprintf(pushDescFmt, mediaType)).Exec()
			ORAS("blob", "push", LayoutFlag, LayoutRef(tmpRoot, pushDigest), blobPath, "-v").
				WithDescription("skip pushing if the blob already exists in the target repo").
				MatchKeyWords("Exists").Exec()
			// validate: oci-layout store will fail fetching blob if index.json doesn't exist
			// ORAS("blob", "fetch", LayoutFlag, tmpRoot, "--output", "-").MatchContent(pushContent).Exec()
		})

		It("should push a blob from a stdin and output the descriptor with specific media-type", func() {
			// prepare
			tmpRoot := GinkgoT().TempDir()
			// test
			mediaType := "test.media"
			ORAS("blob", "push", LayoutFlag, LayoutRef(tmpRoot, pushDigest), "-", "--media-type", mediaType, "--descriptor", "--size", strconv.Itoa(len(pushContent))).
				WithInput(strings.NewReader(pushContent)).
				MatchContent(fmt.Sprintf(pushDescFmt, mediaType)).Exec()
			// validate: oci-layout store will fail fetching blob if index.json doesn't exist
			// ORAS("blob", "fetch", LayoutFlag, tmpRoot, "--output", "-").MatchContent(pushContent).Exec()
		})
	})
})
