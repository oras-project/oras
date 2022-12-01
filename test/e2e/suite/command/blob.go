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
	. "oras.land/oras/test/e2e/internal/utils"
)

var deleteDigest = "sha256:fcde2b2edba56bf408601fb721fe9b5c338d10ee429ea04fae5511b68fbf8fb9"
var deleteDescriptor = "{\"mediaType\":\"application/octet-stream\",\"digest\":\"sha256:fcde2b2edba56bf408601fb721fe9b5c338d10ee429ea04fae5511b68fbf8fb9\",\"size\":3}"
var deleteContent = "bar"
var deleteTag = "foobar"
var repoFmt = fmt.Sprintf("command/blob/%%s/%d/%%s", GinkgoRandomSeed())

var _ = Describe("ORAS beginners:", func() {
	When("running blob command", func() {
		runAndShowPreviewInHelp([]string{"blob"})

		When("running `blob delete`", func() {
			runAndShowPreviewInHelp([]string{"blob", "delete"}, preview_desc, example_desc)

			It("should fail if no blob reference is provided", func() {
				dstRepo := fmt.Sprintf(repoFmt, "delete", "no-ref")
				ORAS("cp", Reference(Host, repo, deleteTag), Reference(Host, dstRepo, deleteTag)).Exec()
				ORAS("blob", "delete").WithFailureCheck().Exec()
				ORAS("blob", "fetch", Reference(Host, dstRepo, deleteDigest), "--output", "-").MatchContent(deleteContent).Exec()
			})

			It("should fail if no confirmation flag and descriptor flag is provided", func() {
				dstRepo := fmt.Sprintf(repoFmt, "delete", "no-confirm")
				ORAS("cp", Reference(Host, repo, deleteTag), Reference(Host, dstRepo, deleteTag)).Exec()
				ORAS("blob", "delete", Reference(Host, dstRepo, deleteDigest), "--descriptor").WithFailureCheck().Exec()
				ORAS("blob", "fetch", Reference(Host, dstRepo, deleteDigest), "--output", "-").MatchContent(deleteContent).Exec()
			})

			It("should fail if the blob reference is not in the form of <name@digest>", func() {
				dstRepo := fmt.Sprintf(repoFmt, "delete", "wrong-ref-form")
				ORAS("blob", "delete", fmt.Sprintf("%s/%s@%s", Host, dstRepo, "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), "--descriptor", "--force").WithFailureCheck().Exec()
				ORAS("blob", "delete", fmt.Sprintf("%s/%s:%s", Host, dstRepo, "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), "--descriptor", "--force").WithFailureCheck().Exec()
				ORAS("blob", "delete", fmt.Sprintf("%s/%s:%s", Host, dstRepo, "test"), "--descriptor", "--force").WithFailureCheck().Exec()
				ORAS("blob", "delete", fmt.Sprintf("%s/%s@%s", Host, dstRepo, "test"), "--descriptor", "--force").WithFailureCheck().Exec()
			})
		})
	})
})

var _ = Describe("Common registry users:", func() {
	When("running `blob delete`", func() {
		It("should delete a blob with interactive confirmation", func() {
			dstRepo := fmt.Sprintf(repoFmt, "delete", "prompt-delete")
			ORAS("cp", Reference(Host, repo, deleteTag), Reference(Host, dstRepo, deleteTag)).Exec()
			toDeleteRef := Reference(Host, dstRepo, deleteDigest)
			ORAS("blob", "delete", toDeleteRef).
				WithInput(strings.NewReader("y")).
				MatchKeyWords("Deleted", toDeleteRef).Exec()
			ORAS("blob", "delete", toDeleteRef).
				WithInput(strings.NewReader("y")).
				WithFailureCheck().
				MatchErrKeyWords("Error:", toDeleteRef, "the specified blob does not exist").Exec()
		})

		It("should delete a blob with confirmation flag and output descriptor", func() {
			dstRepo := fmt.Sprintf(repoFmt, "delete", "prompt-delete")
			ORAS("cp", Reference(Host, repo, deleteTag), Reference(Host, dstRepo, deleteTag)).Exec()
			toDeleteRef := Reference(Host, dstRepo, deleteDigest)
			ORAS("blob", "delete", toDeleteRef, "--force", "--descriptor").MatchContent(deleteDescriptor).Exec()
			ORAS("blob", "delete", toDeleteRef, "--force", "--descriptor").
				WithFailureCheck().
				MatchErrKeyWords("Error:", toDeleteRef, "the specified blob does not exist").Exec()
		})
	})
})
