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
	When("logging out", Ordered, func() {
		It("should use logout command to logout", func() {
			ORAS("logout", Host, "--registry-config", AuthConfigPath).Exec()
		})

		It("should run commands without logging in", func() {
			RunWithoutLogin("attach", Host+"/repo:tag", "-a", "test=true", "--artifact-type", "doc/example")
			ORAS("copy", Host+"/repo:from", Host+"/repo:to", "--from-registry-config", AuthConfigPath, "--to-registry-config", AuthConfigPath).
				ExpectFailure().
				MatchErrKeyWords("Error:", "credential required").
				WithDescription("fail without logging in").Exec()
			RunWithoutLogin("discover", Host+"/repo:tag")
			RunWithoutLogin("push", "-a", "key=value", Host+"/repo:tag")
			RunWithoutLogin("pull", Host+"/repo:tag")
			RunWithoutLogin("manifest", "fetch", Host+"/repo:tag")
			RunWithoutLogin("blob", "delete", Host+"/repo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
			RunWithoutLogin("blob", "push", Host+"/repo", WriteTempFile("blob", "test"))
			RunWithoutLogin("tag", Host+"/repo:tag", "tag1")
			RunWithoutLogin("repo", "ls", Host)
			RunWithoutLogin("repo", "tags", Reference(Host, "repo", ""))
			RunWithoutLogin("manifest", "fetch-config", Host+"/repo:tag")
		})
	})

	When("logging in", func() {
		It("should use basic auth", func() {
			ORAS("login", Host, "-u", Username, "-p", Password, "--registry-config", AuthConfigPath).
				WithTimeOut(20*time.Second).
				MatchContent("Login Succeeded\n").
				MatchErrKeyWords("WARNING", "Using --password via the CLI is insecure", "Use --password-stdin").
				WithDescription("login with username&password flags").Exec()
		})
	})
})

func RunWithoutLogin(args ...string) {
	ORAS(append(args, "--registry-config", AuthConfigPath)...).
		ExpectFailure().
		MatchErrKeyWords("Error:", "credential required").
		WithDescription("fail without logging in").Exec()
}
