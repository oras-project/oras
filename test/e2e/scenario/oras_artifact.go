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
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"oras.land/oras/test/e2e/utils"
	"oras.land/oras/test/e2e/utils/match"
)

var (
	file1       = utils.ImageBlob("foobar/foo1")
	file2       = utils.ImageBlob("foobar/foo2")
	file3       = utils.ImageBlob("foobar/bar")
	config      = utils.ImageBlob("foobar/config")
	emptyConfig = "application/vnd.unknown.config.v1+json"
)

var _ = Context("ORAS user", Focus, Ordered, func() {
	repo := "oras-artifact"
	Describe("logs in", func() {
		When("should succeed with basic auth", func() {
			info := "Login Succeeded\n"
			utils.Exec(match.NewOption(strings.NewReader(PASSWORD), match.NewContent(&info), nil, false),
				"should succeed with username flag and password from stdin",
				"login", utils.Host, "-u", USERNAME, "--password-stdin")
		})
	})

	Describe("pushes image and check", Ordered, func() {
		tag := "image"
		manifestPath := filepath.Join(temp_path, "packed.json")
		When("pushing an image", Ordered, func() {
			pr, pw := io.Pipe()
			AfterAll(func() {
				pr.Close()
				pw.Close()
			})

			status := match.NewStatus([]match.StateKey{
				{Digest: "2c26b46b68ff", Name: file1},
				{Digest: "2c26b46b68ff", Name: file2},
				{Digest: "fcde2b2edba5", Name: file3},
				{Digest: "e3b0c44298fc", Name: "application/vnd.unknown.config.v1+json"},
				// cannot track manifest since created time will be added and digest is unknown
			}, *match.MatchableStatus("push", true), 4)

			utils.Exec(match.NewOption(nil, status, nil, false), "should push files with manifest exported",
				"push", utils.Reference(utils.Host, repo, tag), file1, file2, file3, "-v", "--export-manifest", manifestPath)

			tmp := ""
			s := &tmp
			ginkgo.It("should export the manifest", func() {
				gomega.Expect(manifestPath).Should(gomega.BeAnExistingFile())
				fp, err := os.OpenFile(manifestPath, os.O_RDONLY, 0666)
				gomega.Expect(err).To(gomega.BeNil())
				content, err := io.ReadAll(fp)
				gomega.Expect(err).To(gomega.BeNil())
				*s = string(content)
			})
			utils.Exec(match.SuccessContent(s), "should fetch matching manifest content",
				"manifest", "fetch", utils.Reference(utils.Host, repo, tag))

		})

	})
})
