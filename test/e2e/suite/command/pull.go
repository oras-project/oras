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
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"oras.land/oras-go/v2"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	"oras.land/oras/test/e2e/internal/testdata/multi_arch"
	. "oras.land/oras/test/e2e/internal/utils"
)

var _ = Describe("Remote registry users:", func() {
	When("pulling images from remote registry", func() {
		var (
			repo       = "command/images"
			tag        = "foobar"
			configName = "test.config"
		)

		It("should pull all files in an image to a target folder", func() {
			pullRoot := "pulled"
			tempDir := CopyTestDataToTemp()
			stateKeys := append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageConfigStateKey(configName))
			ORAS("pull", Reference(Host, repo, tag), "-v", "--config", configName, "-o", pullRoot).
				MatchStatus(stateKeys, true, len(stateKeys)).
				WithWorkDir(tempDir).Exec()
			// check config
			configPath := filepath.Join(tempDir, pullRoot, configName)
			Expect(configPath).Should(BeAnExistingFile())
			f, err := os.Open(configPath)
			Expect(err).ShouldNot(HaveOccurred())
			defer f.Close()
			Eventually(gbytes.BufferReader(f)).Should(gbytes.Say("{}"))
			for _, f := range foobar.ImageLayerNames {
				// check layers
				Binary("diff", filepath.Join(tempDir, "foobar", f), filepath.Join(pullRoot, f)).
					WithWorkDir(tempDir).Exec()
			}

			ORAS("pull", Reference(Host, repo, tag), "-v", "-o", pullRoot, "--keep-old-files").
				ExpectFailure().
				WithDescription("fail if overwrite old files are disabled")
		})

		It("should skip config if media type not matching", func() {
			pullRoot := "pulled"
			tempDir := CopyTestDataToTemp()
			stateKeys := append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageConfigStateKey(oras.MediaTypeUnknownConfig))
			ORAS("pull", Reference(Host, repo, tag), "-v", "--config", fmt.Sprintf("%s:%s", configName, "???"), "-o", pullRoot).
				MatchStatus(stateKeys, true, 5).
				WithWorkDir(tempDir).Exec()
			// check config
			Expect(filepath.Join(pullRoot, configName)).ShouldNot(BeAnExistingFile())
			for _, f := range foobar.ImageLayerNames {
				// check layers
				Binary("diff", filepath.Join(tempDir, "foobar", f), filepath.Join(pullRoot, f)).
					WithWorkDir(tempDir).
					WithDescription("should download identical file " + f).Exec()
			}
		})

		It("should pull specific platform", func() {
			ORAS("pull", Reference(Host, repo, "multi"), "--platform", "linux/amd64", "-v", "-o", GinkgoT().TempDir()).
				MatchStatus(multi_arch.ImageStateKey, true, len(multi_arch.ImageStateKey)).Exec()
		})
	})
})
