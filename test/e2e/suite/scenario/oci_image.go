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
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	. "oras.land/oras/test/e2e/internal/utils"
	"oras.land/oras/test/e2e/internal/utils/match"
)

var (
	files = []string{
		"foobar/config.json",
		"foobar/foo1",
		"foobar/foo2",
		"foobar/bar",
	}
)

var _ = Describe("ORAS user", Ordered, func() {
	repo := "oci-image"
	When("logs in", func() {
		It("using basic auth", func() {
			info := "Login Succeeded\n"
			Success("login", Host, "-u", USERNAME, "--password-stdin").
				WithInput(strings.NewReader(PASSWORD)).
				WithTimeOut(30 * time.Second).
				MatchContent(&info).
				Exec("should succeed with username flag and password from stdin")
		})
	})

	When("pushes images and check", func() {
		tag := "image"
		workDir := new(string)
		BeforeAll(func() {
			dir := GinkgoT().TempDir()
			if err := CopyTestData(files, dir); err != nil {
				panic(err)
			}
			*workDir = dir
		})

		It("pushing and pulling an image", func() {
			manifestName := "packed.json"
			Success("push", Reference(Host, repo, tag), "--config", files[0], files[1], files[2], files[3], "-v", "--export-manifest", manifestName).
				MatchStatus([]match.StateKey{
					{Digest: "46b68ac1696c", Name: "application/vnd.unknown.config.v1+json"},
					{Digest: "2c26b46b68ff", Name: files[1]},
					{Digest: "2c26b46b68ff", Name: files[2]},
					{Digest: "fcde2b2edba5", Name: files[3]}}, true, 4).
				WithWorkDir(workDir).
				Exec("should push files with manifest exported")

			exportedContent := new(string)
			ginkgo.By("should export the manifest", func() {
				content, err := os.ReadFile(filepath.Join(*workDir, manifestName))
				gomega.Expect(err).To(gomega.BeNil())
				*exportedContent = string(content)
			})

			Success("manifest", "fetch", Reference(Host, repo, tag)).
				MatchContent(exportedContent).
				Exec("should fetch pushed manifest content")

			pullRoot := "pulled"
			Success("pull", Reference(Host, repo, tag), "-v", "--config", files[0], "-o", pullRoot).
				MatchStatus([]match.StateKey{
					{Digest: "46b68ac1696c", Name: "application/vnd.unknown.config.v1+json"},
					{Digest: "2c26b46b68ff", Name: files[1]},
					{Digest: "2c26b46b68ff", Name: files[2]},
					{Digest: "fcde2b2edba5", Name: files[3]}}, true, 3).
				WithWorkDir(workDir).
				Exec("should pull files with config")

			for _, f := range files {
				Success(filepath.Join(f), filepath.Join(pullRoot, f)).
					WithBinary("diff").
					Exec("should download identical file " + f)
			}
		})

	})
})
