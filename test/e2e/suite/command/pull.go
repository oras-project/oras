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
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "oras.land/oras/test/e2e/internal/utils"
	"oras.land/oras/test/e2e/internal/utils/match"
)

var _ = Describe("Remote registry users:", func() {
	When("pulling images from remote registry", func() {
		var (
			repo  = "command/images"
			tag   = "foobar"
			files = []string{
				"config.default.json",
				"foo1",
				"foo2",
				"bar",
			}
		)

		It("should pull all files in an image to a target folder", func() {
			pullRoot := "pulled"
			tempDir := GinkgoT().TempDir()
			if err := CopyTestData(tempDir); err != nil {
				panic(err)
			}
			ORAS("pull", Reference(Host, repo, tag), "-v", "--config", files[0], "-o", pullRoot).
				MatchStatus([]match.StateKey{
					{Digest: "fd6ed2f36b54", Name: "application/vnd.oci.image.manifest.v1+json"},
					{Digest: "44136fa355b3", Name: files[0]},
					{Digest: "2c26b46b68ff", Name: files[1]},
					{Digest: "2c26b46b68ff", Name: files[2]},
					{Digest: "fcde2b2edba5", Name: files[3]},
				}, true, 5).
				WithWorkDir(tempDir).
				WithDescription("pull files with config").Exec()
			for _, f := range files {
				Binary("diff", filepath.Join(tempDir, "foobar", f), filepath.Join(pullRoot, f)).
					WithWorkDir(tempDir).
					WithDescription("should download identical file " + f).Exec()
			}
		})

		It("should skip config if media type not matching", func() {
			pullRoot := "pulled"
			tempDir := GinkgoT().TempDir()
			if err := CopyTestData(tempDir); err != nil {
				panic(err)
			}
			ORAS("pull", Reference(Host, repo, tag), "-v", "--config", fmt.Sprintf("%s:%s", files[0], "???"), "-o", pullRoot).
				MatchStatus([]match.StateKey{
					{Digest: "fd6ed2f36b54", Name: "application/vnd.oci.image.manifest.v1+json"},
					{Digest: "44136fa355b3", Name: "application/vnd.unknown.config.v1+json"},
					{Digest: "2c26b46b68ff", Name: files[1]},
					{Digest: "2c26b46b68ff", Name: files[2]},
					{Digest: "fcde2b2edba5", Name: files[3]},
				}, true, 5).
				WithWorkDir(tempDir).
				WithDescription("pull files with config").Exec()
			Binary("stat", filepath.Join(pullRoot, files[0])).
				WithWorkDir(tempDir).
				WithFailureCheck().
				MatchErrKeyWords("no such file or directory").Exec()
			for _, f := range files[1:] {
				Binary("diff", filepath.Join(tempDir, "foobar", f), filepath.Join(pullRoot, f)).
					WithWorkDir(tempDir).
					WithDescription("should download identical file " + f).Exec()
			}
		})

		It("should pull specific platform", func() {

		})
	})
})
