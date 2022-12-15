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
	"oras.land/oras/cmd/oras/blob"

	. "oras.land/oras/test/e2e/internal/utils"
)

var pushContent = "test-blob"
var pushLength = strconv.Itoa(len(pushContent))
var pushDigest = "sha256:e1ca41574914ba00e8ed5c8fc78ec8efdfd48941c7e48ad74dad8ada7f2066d8"
var wrongDigest = "sha256:e1ca41574914ba00e8ed5c8fc78ec8efdfd48941c7e48ad74dad8ada7f2066d9"
var pushDescFmt = `{"mediaType":"%s","digest":"sha256:e1ca41574914ba00e8ed5c8fc78ec8efdfd48941c7e48ad74dad8ada7f2066d8","size":9}`
var repoFmt = fmt.Sprintf("command/blob/push/%d/%%s", GinkgoRandomSeed())

var _ = Describe("ORAS beginners:", func() {
	When("running blob command", func() {
		runAndShowPreviewInHelp([]string{"blob"})

		When("running `blob push`", func() {
			runAndShowPreviewInHelp([]string{"blob", "push"}, preview_desc, example_desc)
			It("should fail to read blob content and password from stdin at the same time", func() {
				repo := fmt.Sprintf(repoFmt, "password-stdin")
				ORAS("blob", "push", Reference(Host, repo, ""), "--password-stdin", "-").
					WithFailureCheck().
					MatchTrimmedContent("Error: `-` read file from input and `--password-stdin` read password from input cannot be both used").Exec()
			})
			It("should fail to push a blob from stdin but no blob size provided", func() {
				repo := fmt.Sprintf(repoFmt, "no-size")
				ORAS("blob", "push", Reference(Host, repo, pushDigest), "-").
					WithInput(strings.NewReader(pushContent)).
					WithFailureCheck().
					MatchTrimmedContent("Error: `--size` must be provided if the blob is read from stdin").Exec()
			})

			It("should fail to push a blob from stdin if invalid blob size provided", func() {
				repo := fmt.Sprintf(repoFmt, "invalid-stdin-size")
				blob.Cmd().ExecuteC()
				ORAS("blob", "push", Reference(Host, repo, pushDigest), "-", "--size", "3").
					WithInput(strings.NewReader(pushContent)).WithFailureCheck().
					Exec()
			})

			It("should fail to push a blob from stdin if invalid digest provided", func() {
				repo := fmt.Sprintf(repoFmt, "invalid-stdin-digest")
				ORAS("blob", "push", Reference(Host, repo, wrongDigest), "-", "--size", pushLength).
					WithInput(strings.NewReader(pushContent)).WithFailureCheck().
					Exec()
			})

			It("should fail to push a blob from file if invalid blob size provided", func() {
				repo := fmt.Sprintf(repoFmt, "invalid-file-digest")
				blobPath := WriteTempFile("blob", pushContent)
				ORAS("blob", "push", Reference(Host, repo, pushDigest), blobPath, "--size", "3").
					WithFailureCheck().
					Exec()
			})

			It("should fail to push a blob from file if invalid digest provided", func() {
				repo := fmt.Sprintf(repoFmt, "invalid-stdin-size")
				blobPath := WriteTempFile("blob", pushContent)
				ORAS("blob", "push", Reference(Host, repo, wrongDigest), blobPath, "--size", string(rune(len(pushContent)))).
					WithInput(strings.NewReader(pushContent)).WithFailureCheck().
					Exec()
			})

			It("should fail if no reference is provided", func() {
				ORAS("blob", "push").WithFailureCheck().Exec()
			})
		})

		When("running `blob fetch`", func() {
			runAndShowPreviewInHelp([]string{"blob", "fetch"}, preview_desc, example_desc)

			It("should call sub-commands with aliases", func() {
				ORAS("blob", "get", "--help").
					MatchKeyWords("[Preview] Fetch", preview_desc, example_desc).
					Exec()
			})
			It("should have flag for prettifying JSON output", func() {
				ORAS("blob", "get", "--help").
					MatchKeyWords("--pretty", "prettify JSON").
					Exec()
			})

			It("should fail if neither output path nor descriptor flag are not provided", func() {
				ORAS("blob", "fetch", Reference(Host, repo, "sha256:2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae")).
					WithFailureCheck().Exec()
			})

			It("should fail if no digest provided", func() {
				ORAS("blob", "fetch", Reference(Host, repo, "")).
					WithFailureCheck().Exec()
			})

			It("should fail if provided digest doesn't existed", func() {
				ORAS("blob", "fetch", Reference(Host, repo, "sha256:2aaa2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a")).
					WithFailureCheck().Exec()
			})

			It("should fail if output path points to stdout and descriptor flag is provided", func() {
				ORAS("blob", "fetch", Reference(Host, repo, ""), "--descriptor", "--output", "-").
					WithFailureCheck().Exec()
			})

			It("should fail if no reference is provided", func() {
				ORAS("blob", "fetch").WithFailureCheck().Exec()
			})
		})
	})
})

var _ = Describe("Common registry users:", func() {
	When("running `blob push`", func() {
		It("should push a blob from a file and output the descriptor with specific media-type", func() {
			mediaType := "test.media"
			repo := fmt.Sprintf(repoFmt, "blob-file-media-type")
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
			repo := fmt.Sprintf(repoFmt, "blob-file-media-type")
			ORAS("blob", "push", Reference(Host, repo, pushDigest), "-", "--media-type", mediaType, "--descriptor", "--size", pushLength).
				WithInput(strings.NewReader(pushContent)).
				MatchContent(fmt.Sprintf(pushDescFmt, mediaType)).Exec()
			ORAS("blob", "fetch", Reference(Host, repo, pushDigest), "--output", "-").MatchContent(pushContent).Exec()
		})
	})

	var blobDigest = "sha256:2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae"
	var blobContent = "foo"
	var blobDescriptor = `{"mediaType":"application/octet-stream","digest":"sha256:2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae","size":3}`
	When("running `blob fetch`", func() {
		It("should fetch blob descriptor ", func() {
			ORAS("blob", "fetch", Reference(Host, repo, blobDigest), "--descriptor").
				MatchContent(blobDescriptor).Exec()
		})
		It("should fetch blob content and output to stdout", func() {
			ORAS("blob", "fetch", Reference(Host, repo, blobDigest), "--output", "-").
				MatchContent(blobContent).Exec()
		})
		It("should fetch blob content and output to a file", func() {
			tempDir := GinkgoT().TempDir()
			contentPath := filepath.Join(tempDir, "fetched")
			ORAS("blob", "fetch", Reference(Host, repo, blobDigest), "--output", contentPath).
				WithWorkDir(tempDir).Exec()
			MatchFile(contentPath, blobContent, DefaultTimeout)
		})
		It("should fetch blob descriptor and output content to a file", func() {
			tempDir := GinkgoT().TempDir()
			contentPath := filepath.Join(tempDir, "fetched")
			ORAS("blob", "fetch", Reference(Host, repo, blobDigest), "--output", contentPath, "--descriptor").
				MatchContent(blobDescriptor).
				WithWorkDir(tempDir).Exec()
			MatchFile(contentPath, blobContent, DefaultTimeout)
		})
	})
})
