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
	. "oras.land/oras/test/e2e/internal/utils"
)

const (
	USERNAME         = "hello"
	PASSWORD         = "oras-test"
	AUTH_CONFIG_PATH = "test.config"
)

var _ = Describe("ORAS user", Ordered, func() {
	Context("auth", func() {
		info := "Login Succeeded\n"
		When("using basic auth", func() {
			Success("login", HOST, "-u", USERNAME, "-p", PASSWORD, "--registry-config", AUTH_CONFIG_PATH).
				MatchContent(&info).
				WithStderrKeyWords("WARNING", "Using --password via the CLI is insecure", "Use --password-stdin").Exec("should succeed with username&password flags")
		})
	})

	Context("logs out", func() {
		When("using logout command", func() {
			Success("logout", HOST, "--registry-config", AUTH_CONFIG_PATH).
				Exec("should log out")
		})
	})

	Context("runs commands without login", func() {
		whenRunWithoutLogin("attach", HOST+"/repo:tag", "-a", "test=true", "--artifact-type", "doc/example")
		whenRunWithoutLogin("copy", HOST+"/repo:from", HOST+"/repo:to")
		whenRunWithoutLogin("discover", HOST+"/repo:tag")
		whenRunWithoutLogin("push", "-a", "key=value", HOST+"/repo:tag")
		whenRunWithoutLogin("pull", HOST+"/repo:tag")

		whenRunWithoutLogin("manifest", "fetch", HOST+"/repo:tag")
	})
})

func whenRunWithoutLogin(args ...string) {
	When("running "+args[0]+" command", func() {
		Error(append(args, "--registry-config", AUTH_CONFIG_PATH)...).
			WithStderrKeyWords("Error:", "credential required").
			Exec("should failed")
	})
}
