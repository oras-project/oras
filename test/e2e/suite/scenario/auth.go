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
	// customerize your own basic auth file via `htpasswd -cBb <file_name> <user_name> <password>`
	USERNAME         = "hello"
	PASSWORD         = "oras-test"
	AUTH_CONFIG_PATH = "test.config"
)

var _ = Describe("ORAS User", Ordered, func() {
	When("logging in", func() {
		info := "Login Succeeded\n"
		It("uses basic auth", func() {
			Success("login", Host, "-u", USERNAME, "-p", PASSWORD, "--registry-config", AUTH_CONFIG_PATH).
				MatchContent(&info).
				MatchErrKeyWords("WARNING", "Using --password via the CLI is insecure", "Use --password-stdin").Exec("should succeed with username&password flags")
		})
	})

	When("logging out", func() {
		It("using logout command", func() {
			Success("logout", Host, "--registry-config", AUTH_CONFIG_PATH).
				Exec("should log out")
		})
	})

	When("running commands without login", func() {
		RunWithoutLogin("attach", Host+"/repo:tag", "-a", "test=true", "--artifact-type", "doc/example")
		RunWithoutLogin("copy", Host+"/repo:from", Host+"/repo:to")
		RunWithoutLogin("discover", Host+"/repo:tag")
		RunWithoutLogin("push", "-a", "key=value", Host+"/repo:tag")
		RunWithoutLogin("pull", Host+"/repo:tag")
		RunWithoutLogin("manifest", "fetch", Host+"/repo:tag")
	})
})

func RunWithoutLogin(args ...string) {
	It("runs "+args[0]+" command", func() {
		Error(append(args, "--registry-config", AUTH_CONFIG_PATH)...).
			MatchErrKeyWords("Error:", "credential required").
			Exec("should fail without logging in")
	})
}
