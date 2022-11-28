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
})
