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
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"oras.land/oras-go/v2"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	. "oras.land/oras/test/e2e/internal/utils"
)

var _ = Describe("Common registry user", func() {
	When("not logged in", func() {
		It("should run commands without logging in", func() {
			authConfigPath := filepath.Join(GinkgoT().TempDir(), "auth.config")
			RunWithoutLogin("attach", ZOTHost+"/repo:tag", "-a", "test=true", "--artifact-type", "doc/example", "--registry-config", authConfigPath)
			RunWithoutLogin("copy", ZOTHost+"/repo:from", ZOTHost+"/repo:to", "--from-registry-config", authConfigPath, "--to-registry-config", authConfigPath)
			RunWithoutLogin("discover", ZOTHost+"/repo:tag", "--registry-config", authConfigPath)
			RunWithoutLogin("push", "-a", "key=value", ZOTHost+"/repo:tag", "--registry-config", authConfigPath)
			RunWithoutLogin("pull", ZOTHost+"/repo:tag", "--registry-config", authConfigPath)
			RunWithoutLogin("manifest", "fetch", ZOTHost+"/repo:tag", "--registry-config", authConfigPath)
			RunWithoutLogin("blob", "delete", ZOTHost+"/repo@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "--registry-config", authConfigPath)
			RunWithoutLogin("blob", "push", ZOTHost+"/repo", WriteTempFile("blob", "test"), "--registry-config", authConfigPath)
			RunWithoutLogin("tag", ZOTHost+"/repo:tag", "tag1", "--registry-config", authConfigPath)
			RunWithoutLogin("resolve", ZOTHost+"/repo:tag", "--registry-config", authConfigPath)
			RunWithoutLogin("repo", "ls", ZOTHost, "--registry-config", authConfigPath)
			RunWithoutLogin("repo", "tags", RegistryRef(ZOTHost, "repo", ""), "--registry-config", authConfigPath)
			RunWithoutLogin("manifest", "fetch-config", ZOTHost+"/repo:tag", "--registry-config", authConfigPath)
		})
	})

	When("logging in", func() {
		tmpConfigName := "test.config"
		It("should succeed to use basic auth", func() {
			ORAS("login", ZOTHost, "-u", Username, "-p", Password, "--registry-config", filepath.Join(GinkgoT().TempDir(), tmpConfigName)).
				WithTimeOut(20*time.Second).
				MatchContent("Login Succeeded\n").
				MatchErrKeyWords("WARNING", "Using --password via the CLI is insecure", "Use --password-stdin").Exec()
		})

		It("should show detailed error description if no argument provided", func() {
			err := ORAS("login").ExpectFailure().Exec().Err
			Expect(err).Should(gbytes.Say("Error"))
			Expect(err).Should(gbytes.Say("\nUsage: oras login"))
			Expect(err).Should(gbytes.Say("\n"))
			Expect(err).Should(gbytes.Say(`Run "oras login -h"`))
		})

		It("should fail if no username input", func() {
			ORAS("login", ZOTHost, "--registry-config", filepath.Join(GinkgoT().TempDir(), tmpConfigName)).
				WithTimeOut(20 * time.Second).
				WithInput(strings.NewReader("")).
				MatchKeyWords("username:").
				ExpectFailure().
				Exec()
		})

		It("should fail if no password input", func() {
			ORAS("login", ZOTHost, "--registry-config", filepath.Join(GinkgoT().TempDir(), tmpConfigName)).
				WithTimeOut(20*time.Second).
				MatchKeyWords("Username: ", "Password: ").
				WithInput(strings.NewReader(fmt.Sprintf("%s\n", Username))).ExpectFailure().Exec()
		})

		It("should fail if password is empty", func() {
			ORAS("login", ZOTHost, "--registry-config", filepath.Join(GinkgoT().TempDir(), tmpConfigName)).
				WithTimeOut(20*time.Second).
				MatchKeyWords("Username: ", "Password: ").
				MatchErrKeyWords("Error: password required").
				WithInput(strings.NewReader(fmt.Sprintf("%s\n\n", Username))).ExpectFailure().Exec()
		})

		It("should fail if password is wrong with registry error prefix", func() {
			ORAS("login", ZOTHost, "--registry-config", filepath.Join(GinkgoT().TempDir(), tmpConfigName)).
				WithTimeOut(20*time.Second).
				MatchKeyWords("Username: ", "Password: ").
				MatchErrKeyWords("Error response from registry: ").
				WithInput(strings.NewReader(fmt.Sprintf("%s\n???\n", Username))).ExpectFailure().Exec()
		})

		It("should fail if no token input", func() {
			ORAS("login", ZOTHost, "--registry-config", filepath.Join(GinkgoT().TempDir(), tmpConfigName)).
				WithTimeOut(20*time.Second).
				MatchKeyWords("Username: ", "Token: ").
				WithInput(strings.NewReader("\n")).ExpectFailure().Exec()
		})

		It("should fail if token is empty", func() {
			ORAS("login", ZOTHost, "--registry-config", filepath.Join(GinkgoT().TempDir(), tmpConfigName)).
				WithTimeOut(20*time.Second).
				MatchKeyWords("Username: ", "Token: ").
				MatchErrKeyWords("Error: token required").
				WithInput(strings.NewReader("\n\n")).ExpectFailure().Exec()
		})

		It("should succeed to use prompted input", func() {
			ORAS("login", ZOTHost, "--registry-config", filepath.Join(GinkgoT().TempDir(), tmpConfigName)).
				WithTimeOut(20*time.Second).
				WithInput(strings.NewReader(fmt.Sprintf("%s\n%s\n", Username, Password))).
				MatchKeyWords("Username: ", "Password: ", "Login Succeeded\n").Exec()
		})
	})

	When("using legacy config", func() {
		var LegacyConfigPath = filepath.Join(TestDataRoot, LegacyConfigName)
		It("should succeed to copy", func() {
			src := RegistryRef(ZOTHost, ArtifactRepo, foobar.Tag)
			dst := RegistryRef(ZOTHost, fmt.Sprintf("command/auth/%d/copy", GinkgoRandomSeed()), foobar.Tag)
			foobarStates := append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageConfigStateKey(oras.MediaTypeUnknownConfig))
			ORAS("cp", src, dst, "-v", "--from-registry-config", LegacyConfigPath, "--to-registry-config", LegacyConfigPath).MatchStatus(foobarStates, true, len(foobarStates)).Exec()
		})
	})
})

func RunWithoutLogin(args ...string) {
	ORAS(args...).ExpectFailure().
		MatchErrKeyWords("Error:", "credential required").
		WithDescription("fail without logging in").Exec()
}
