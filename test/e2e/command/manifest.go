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
	"oras.land/oras/test/e2e/step"
	"oras.land/oras/test/e2e/utils"
	"oras.land/oras/test/e2e/utils/match"
)

const (
	preview_desc = "** This command is in preview and under development. **"
	example_desc = "\nExample - "
)

var _ = Context("ORAS beginners", func() {
	Describe("run manifest command", func() {
		step.RunAndShowPreviewInHelp([]string{"manifest"})
		step.RunAndShowPreviewInHelp([]string{"manifest", "fetch"}, preview_desc, example_desc)

		When("call sub-commands with aliases", func() {
			utils.Exec(match.SuccessKeywords("[Preview] Fetch", preview_desc, example_desc),
				"should succeed",
				"manifest", "get", "--help")
		})
		When("fetching manifest with no artifact reference provided", func() {
			utils.Exec(match.ErrorKeywords("Error:"), "should fail",
				"manifest", "fetch",
			)
		})
	})
})
