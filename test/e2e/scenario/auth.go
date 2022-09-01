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
	"oras.land/oras/test/e2e/step"
	"oras.land/oras/test/e2e/utils"
	"oras.land/oras/test/e2e/utils/match"
)

const (
	USERNAME = "hello"
	PASSWORD = "oras-test"
)

var _ = Context("ORAS user", Ordered, func() {

	Describe("logs in", func() {
		When("should succeed with basic auth", func() {
			utils.Exec("should succeed with username&password flags",
				[]string{"login", utils.Host, "-u", USERNAME, "-p", PASSWORD},
				match.NewResult(nil, match.Content("Login Succeeded\n"), match.Keyword([]string{"WARNING", "Using --password via the CLI is insecure", "Use --password-stdin"}), false))

			utils.Exec("should succeed with username flag and password from stdin",
				[]string{"login", utils.Host, "-u", USERNAME, "--password-stdin"},
				match.NewResult(strings.NewReader(PASSWORD), match.Content("Login Succeeded\n"), nil, false))
		})
	})

	Describe("logs out", func() {
		When("should succeed", func() {
			utils.Exec("should logout",
				[]string{"logout", utils.Host},
				match.NewResult(nil, nil, nil, false))
		})
	})

	Describe("runs commands without login", func() {
		step.WhenRunWithoutLogin([]string{"attach", utils.Host + "/repo:tag", "-a", "test=true", "--artifact-type", "doc/example"})
		step.WhenRunWithoutLogin([]string{"copy", utils.Host + "/repo:from", utils.Host + "/repo:to"})
		step.WhenRunWithoutLogin([]string{"discover", utils.Host + "/repo:tag"})
		step.WhenRunWithoutLogin([]string{"push", "-a", "key=value", utils.Host + "/repo:tag"})
		step.WhenRunWithoutLogin([]string{"pull", utils.Host + "/repo:tag"})

		step.WhenRunWithoutLogin([]string{"manifest", "fetch", utils.Host + "/repo:tag"})
	})
})
