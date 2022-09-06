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

const (
	USERNAME = "hello"
	PASSWORD = "oras-test"
)

var _ = Context("ORAS user", Ordered, func() {
	Describe("auth", func() {
		info := "Login Succeeded\n"
		When("should succeed with basic auth", func() {
			utils.Exec(match.NewOption(nil, match.NewContent(&info), match.Keywords([]string{"WARNING", "Using --password via the CLI is insecure", "Use --password-stdin"}), false),
				"should succeed with username&password flags",
				"login", utils.Host, "-u", USERNAME, "-p", PASSWORD)

			utils.Exec(match.NewOption(strings.NewReader(PASSWORD), match.NewContent(&info), nil, false),
				"should succeed with username flag and password from stdin",
				"login", utils.Host, "-u", USERNAME, "--password-stdin")
		})
	})

	Describe("logs out", func() {
		When("should succeed", func() {
			utils.Exec(&match.Success, "should logout", "logout", utils.Host)
		})
	})

	Describe("runs commands without login", func() {
		whenRunWithoutLogin("attach", utils.Host+"/repo:tag", "-a", "test=true", "--artifact-type", "doc/example")
		whenRunWithoutLogin("copy", utils.Host+"/repo:from", utils.Host+"/repo:to")
		whenRunWithoutLogin("discover", utils.Host+"/repo:tag")
		whenRunWithoutLogin("push", "-a", "key=value", utils.Host+"/repo:tag")
		whenRunWithoutLogin("pull", utils.Host+"/repo:tag")

		whenRunWithoutLogin("manifest", "fetch", utils.Host+"/repo:tag")
	})
})

func whenRunWithoutLogin(args ...string) {
	When("running "+args[0]+" command", func() {
		utils.Exec(match.ErrorKeywords("Error:", "credential required"),
			"should failed",
			args...)
	})
}
