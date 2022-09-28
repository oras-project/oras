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
	wd string
)

var _ = Describe("ORAS user", Focus, Ordered, func() {
	BeforeAll(func() {
		wd = GinkgoT().TempDir()
		if err := CopyTestData(files, wd); err != nil {
			panic(err)
		}
	})

	repo := "oci-image"
	Context("logs in", func() {
		When("using basic auth", func() {
			info := "Login Succeeded\n"
			Exec(Success().WithInput(strings.NewReader(PASSWORD)).WithContent(&info),
				"should succeed with username flag and password from stdin",
				"login", HOST, "-u", USERNAME, "--password-stdin")
		})
	})

	Context("pushes images and check", Ordered, func() {
		tag := "image"
		When("pushing and pulling an image", Ordered, func() {
			manifestPath := filepath.Join(wd, "packed.json")
			tmp := ""
			s := &tmp
			paths := []string{
				filepath.Join(wd, files[0]),
				filepath.Join(wd, files[1]),
				filepath.Join(wd, files[2]),
				filepath.Join(wd, files[3]),
			}

			Exec(Success().WithStatus([]match.StateKey{
				{Digest: "44136fa355b3", Name: "application/vnd.unknown.config.v1+json"},
				{Digest: "2c26b46b68ff", Name: paths[1]},
				{Digest: "2c26b46b68ff", Name: paths[2]},
				{Digest: "fcde2b2edba5", Name: paths[3]},
				// cannot track manifest since created time will be added and digest is unknown
			}, "push", true, 4), "should push files with manifest exported",
				"push", Reference(HOST, repo, tag), paths[1], paths[2], paths[3], "--config", paths[0], "-v", "--export-manifest", manifestPath)

			ginkgo.It("should export the manifest", func() {
				content, err := os.ReadFile(manifestPath)
				gomega.Expect(err).To(gomega.BeNil())
				*s = string(content)
			})

			Exec(Success().WithContent(s), "should fetch pushed manifest content",
				"manifest", "fetch", Reference(HOST, repo, tag))

			ginkgo.It("should move pushed", func() {
				err := os.Rename(wd, wd+"-pushed")
				gomega.Expect(err).To(gomega.BeNil())
			})

			// configName := "config.json"
			Exec(Success().WithStatus([]match.StateKey{
				{Digest: "44136fa355b3", Name: "application/vnd.unknown.config.v1+json"},
				{Digest: "2c26b46b68ff", Name: paths[1]},
				{Digest: "2c26b46b68ff", Name: paths[2]},
				{Digest: "fcde2b2edba5", Name: paths[3]},
				// cannot track manifest since created time will be added and digest is unknown
			}, "pull", true, 2), "should pull files with config",
				"pull", Reference(HOST, repo, tag), "-v", "--config", paths[0], "-o", wd)
			for i := range paths {
				ginkgo.It("should download file "+paths[i], func() {
					got, err := os.ReadFile(paths[i])
					gomega.Expect(err).To(gomega.BeNil())

					want, err := os.ReadFile(filepath.Join(wd+"-pushed", files[i]))
					gomega.Expect(err).To(gomega.BeNil())
					gomega.Expect(string(got)).To(gomega.Equal(string(want)))
				})
			}
		})

	})
})
