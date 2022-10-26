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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "oras.land/oras/test/e2e/internal/utils"
)

var _ = Describe("Common registry user", Ordered, func() {
	When("logging in", func() {
		It("should use basic auth", func() {
			ORAS("login", Host, "-u", USERNAME, "-p", PASSWORD, "--registry-config", AUTH_CONFIG_PATH).
				WithTimeOut(20*time.Second).
				MatchContent("Login Succeeded\n").
				MatchErrKeyWords("WARNING", "Using --password via the CLI is insecure", "Use --password-stdin").
				WithDescription("login with username&password flags").Exec()
		})
	})

	When("logging out", Ordered, func() {
		It("should use logout command to logout", func() {
			ORAS("logout", Host, "--registry-config", AUTH_CONFIG_PATH).Exec()
		})

		It("should run commands without logging in", func() {
			RunWithoutLogin("attach", Host+"/repo:tag", "-a", "test=true", "--artifact-type", "doc/example")
			RunWithoutLogin("copy", Host+"/repo:from", Host+"/repo:to")
			RunWithoutLogin("discover", Host+"/repo:tag")
			RunWithoutLogin("push", "-a", "key=value", Host+"/repo:tag")
			RunWithoutLogin("pull", Host+"/repo:tag")
			RunWithoutLogin("manifest", "fetch", Host+"/repo:tag")
		})
	})
})

func RunWithoutLogin(args ...string) {
	ORAS(append(args, "--registry-config", AUTH_CONFIG_PATH)...).
		WithFailureCheck().
		MatchErrKeyWords("Error:", "credential required").
		WithDescription("fail without logging in").Exec()
}
