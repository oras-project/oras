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

package scenario

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	"oras.land/oras/test/e2e/utils"
	"oras.land/oras/test/e2e/utils/match"
)

var _ = Context("ORAS user", Ordered, func() {
	Describe("logs in", func() {
		When("should succeed with basic auth", func() {
			utils.Exec(match.NewOption(strings.NewReader(PASSWORD), match.Content("Login Succeeded\n"), nil, false),
				"should succeed with username flag and password from stdin",
				"login", utils.Host, "-u", USERNAME, "--password-stdin")
		})
	})

	Describe("runs commands without login", Ordered, func() {
		When("pushing an image", func() {
			match.NewStatus([]match.StateKey{
				{}
			})

			utils.Exec(match.NewOption(nil,
			utils.Exec(match.NewOption(nil, match.Content("Login Succeeded\n"), nil, false),
				"should succeed with username flag and password from stdin",
				"login", utils.Host, "-u", USERNAME, "--password-stdin")
		})
		whenRunWithoutLogin("attach", utils.Host+"/repo:tag", "-a", "test=true", "--artifact-type", "doc/example")
		whenRunWithoutLogin("copy", utils.Host+"/repo:from", utils.Host+"/repo:to")
		whenRunWithoutLogin("discover", utils.Host+"/repo:tag")
		whenRunWithoutLogin("push", "-a", "key=value", utils.Host+"/repo:tag")
		whenRunWithoutLogin("pull", utils.Host+"/repo:tag")

		whenRunWithoutLogin("manifest", "fetch", utils.Host+"/repo:tag")
	})
})
