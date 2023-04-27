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
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	. "oras.land/oras/test/e2e/internal/utils"
)

var _ = Describe("OCI image user:", Ordered, func() {
	repo := "scenario/oci-image"
	files := append([]string{foobar.FileConfigName}, foobar.FileLayerNames...)
	statusKeys := append(foobar.FileStateKeys, foobar.FileConfigStateKey)
	When("pushing images and check", func() {
		tag := "image"
		var tempDir string
		BeforeAll(func() {
			tempDir = PrepareTempFiles()
		})

		It("should push and pull an image", func() {
			manifestName := "packed.json"
			ORAS("push", RegistryRef(Host, repo, tag), "--config", files[0], files[1], files[2], files[3], "-v", "--export-manifest", manifestName).
				MatchStatus(statusKeys, true, 4).
				WithWorkDir(tempDir).
				WithDescription("push files with manifest exported").Exec()

			fetched := ORAS("manifest", "fetch", RegistryRef(Host, repo, tag)).
				WithDescription("fetch pushed manifest content").Exec().Out.Contents()
			MatchFile(filepath.Join(tempDir, manifestName), string(fetched), DefaultTimeout)

			pullRoot := "pulled"
			ORAS("pull", RegistryRef(Host, repo, tag), "-v", "--config", files[0], "-o", pullRoot).
				MatchStatus(statusKeys, true, 3).
				WithWorkDir(tempDir).
				WithDescription("pull files with config").Exec()

			for _, f := range files {
				Binary("diff", filepath.Join(f), filepath.Join(pullRoot, f)).
					WithWorkDir(tempDir).
					WithDescription("should download identical file " + f).Exec()
			}
		})

	})
})
