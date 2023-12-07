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
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	. "oras.land/oras/test/e2e/internal/utils"
)

var _ = Describe("Common registry user", Ordered, func() {
	When("logging out", Ordered, func() {
		It("should use logout command to logout", func() {
			ORAS("logout", ZOTHost, "--registry-config", AuthConfigPath).Exec()
		})

		It("should run commands without logging in", func() {
			RunWithoutLogin("attach", ZOTHost+"/repo:tag", "-a", "test=true", "--artifact-type", "doc/example")
			ORAS("copy", ZOTHost+"/repo:from", ZOTHost+"/repo:to", "--from-registry-config", AuthConfigPath, "--to-registry-config", AuthConfigPath).
				ExpectFailure().
				MatchErrKeyWords("Error:", "credential required").
				WithDescription("fail without logging in").Exec()
			RunWithoutLogin("discover", ZOTHost+"/repo:tag")
			RunWithoutLogin("push", "-a", "key=value", ZOTHost+"/repo:tag")
			RunWithoutLogin("pull", ZOTHost+"/repo:tag")
			RunWithoutLogin("manifest", "fetch", ZOTHost+"/repo:tag")
			RunWithoutLogin("blob", "delete", ZOTHost+"/repo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
			RunWithoutLogin("blob", "push", ZOTHost+"/repo", WriteTempFile("blob", "test"))
			RunWithoutLogin("tag", ZOTHost+"/repo:tag", "tag1")
			RunWithoutLogin("resolve", ZOTHost+"/repo:tag")
			RunWithoutLogin("repo", "ls", ZOTHost)
			RunWithoutLogin("repo", "tags", RegistryRef(ZOTHost, "repo", ""))
			RunWithoutLogin("manifest", "fetch-config", ZOTHost+"/repo:tag")
		})
	})

	When("logging in", func() {
		It("should use basic auth", func() {
			ORAS("login", ZOTHost, "-u", Username, "-p", Password, "--registry-config", AuthConfigPath).
				WithTimeOut(20*time.Second).
				MatchContent("Login Succeeded\n").
				MatchErrKeyWords("WARNING", "Using --password via the CLI is insecure", "Use --password-stdin").Exec()
		})

		It("should show detailed error description if no argument provided", func() {
			err := ORAS("login").ExpectFailure().Exec().Err
			Expect(err).Should(gbytes.Say("Error"))
			Expect(err).Should(gbytes.Say("\nUsage: login"))
			Expect(err).Should(gbytes.Say("\n"))
			Expect(err).Should(gbytes.Say(`Run "oras login -h"`))
		})

		It("should fail if no username input", func() {
			ORAS("login", ZOTHost, "--registry-config", AuthConfigPath).
				WithTimeOut(20 * time.Second).
				WithInput(strings.NewReader("")).
				MatchKeyWords("username:").
				ExpectFailure().
				Exec()
		})

		It("should fail if no password input", func() {
			ORAS("login", ZOTHost, "--registry-config", AuthConfigPath).
				WithTimeOut(20*time.Second).
				MatchKeyWords("Username: ", "Password: ").
				WithInput(strings.NewReader(fmt.Sprintf("%s\n", Username))).ExpectFailure().Exec()
		})

		It("should fail if password is empty", func() {
			ORAS("login", ZOTHost, "--registry-config", AuthConfigPath).
				WithTimeOut(20*time.Second).
				MatchKeyWords("Username: ", "Password: ").
				MatchErrKeyWords("Error: password required").
				WithInput(strings.NewReader(fmt.Sprintf("%s\n\n", Username))).ExpectFailure().Exec()
		})

		It("should fail if no token input", func() {
			ORAS("login", ZOTHost, "--registry-config", AuthConfigPath).
				WithTimeOut(20*time.Second).
				MatchKeyWords("Username: ", "Token: ").
				WithInput(strings.NewReader("\n")).ExpectFailure().Exec()
		})

		It("should fail if token is empty", func() {
			ORAS("login", ZOTHost, "--registry-config", AuthConfigPath).
				WithTimeOut(20*time.Second).
				MatchKeyWords("Username: ", "Token: ").
				MatchErrKeyWords("Error: token required").
				WithInput(strings.NewReader("\n\n")).ExpectFailure().Exec()
		})

		It("should use prompted input", func() {
			ORAS("login", ZOTHost, "--registry-config", AuthConfigPath).
				WithTimeOut(20*time.Second).
				WithInput(strings.NewReader(fmt.Sprintf("%s\n%s\n", Username, Password))).
				MatchKeyWords("Username: ", "Password: ", "Login Succeeded\n").Exec()
		})
	})
})

func RunWithoutLogin(args ...string) {
	ORAS(append(args, "--registry-config", AuthConfigPath)...).
		ExpectFailure().
		MatchErrKeyWords("Error:", "credential required").
		WithDescription("fail without logging in").Exec()
}
