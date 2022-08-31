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
	. "github.com/onsi/ginkgo/v2"
	"oras.land/oras/test/e2e/step"
	"oras.land/oras/test/e2e/utils"
)

const (
	USERNAME = "hello"
	PASSWORD = "oras-test"
)

var _ = Context("ORAS user", Ordered, func() {
	Describe("runs commands without login", func() {
		step.WhenLoginWithoutLogin([]string{"attach", utils.Host + "/repo:tag", "-a", "test=true", "--artifact-type", "doc/example"})
		step.WhenLoginWithoutLogin([]string{"copy", utils.Host + "/repo:from", utils.Host + "/repo:to"})
		step.WhenLoginWithoutLogin([]string{"discover", utils.Host + "/repo:tag"})
		step.WhenLoginWithoutLogin([]string{"push", "-a", "key=value", utils.Host + "/repo:tag"})
		step.WhenLoginWithoutLogin([]string{"pull", utils.Host + "/repo:tag"})

		step.WhenLoginWithoutLogin([]string{"manifest", "fetch", utils.Host + "/repo:tag"})
	})

	Describe("logs in", func() {
		When("should succeed with basic auth", func() {
			utils.ExecAndMatchOut("should succeed with ",
				[]string{"login", utils.Host, "-u", USERNAME, "-p", PASSWORD},
				"Login Succeeded")
		})
	})
})
