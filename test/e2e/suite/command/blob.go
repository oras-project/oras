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

	. "github.com/onsi/ginkgo/v2"
	. "oras.land/oras/test/e2e/internal/utils"
)

var deleteDigest = "sha256:2ef548696ac7dd66ef38aab5cc8fc5cc1fb637dfaedb3a9afc89bf16db9277e1"
var deleteTag = "foobar"
var repoFmt = fmt.Sprintf("command/blob/delete/%d/%%s", GinkgoRandomSeed())

var _ = Describe("ORAS beginners:", func() {
	When("running blob command", func() {
		runAndShowPreviewInHelp([]string{"blob"})

		When("running `blob delete`", func() {
			runAndShowPreviewInHelp([]string{"blob", "delete"}, preview_desc, example_desc)

			It("should fail if no blob reference is provided", func() {
				dstRepo := fmt.Sprintf(repoFmt, "no-ref")
				ORAS("cp", Reference(Host, repo, deleteTag), dstRepo).Exec()
				ORAS("blob", "delete").Exec()
				ORAS("blob", "fetch", Reference(Host, dstRepo, deleteDigest)).MatchContent("hello").Exec()
			})
		})
	})
})

var _ = Describe("Common registry users:", func() {
	When("running `blob delete`", func() {
	})
})
