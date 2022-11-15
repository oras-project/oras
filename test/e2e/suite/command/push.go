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

package command

import (
	. "github.com/onsi/ginkgo/v2"
	. "oras.land/oras/test/e2e/internal/utils"
	"oras.land/oras/test/e2e/internal/utils/match"
)

var _ = Describe("Remote registry users:", func() {
	files := []string{
		"foobar/config.json",
		"foobar/bar",
	}
	statusKeys := []match.StateKey{
		{Digest: "46b68ac1696c", Name: "application/vnd.unknown.config.v1+json"},
		{Digest: "fcde2b2edba5", Name: files[1]},
	}

	repo := "command/push"
	var tempDir string
	BeforeAll(func() {
		tempDir = GinkgoT().TempDir()
		if err := CopyTestData(tempDir); err != nil {
			panic(err)
		}
	})

	When("pushing OCI image", func() {
		It("should push files without customized media types", func() {
			tag := "basic"
			ORAS("push", Reference(Host, repo, "basic"), files[1], "-v").
				MatchStatus(statusKeys, true, 3).
				WithWorkDir(tempDir).Exec()
			fetched := ORAS("manifest", "fetch", Reference(Host, repo, tag)).Exec().Out
			Binary("jq").
				MatchContent("test").
				WithInput(fetched).Exec()
		})
	})
})
