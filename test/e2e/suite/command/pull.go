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
	"regexp"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"oras.land/oras-go/v2"
	"oras.land/oras/test/e2e/internal/testdata/feature"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	"oras.land/oras/test/e2e/internal/testdata/multi_arch"
	. "oras.land/oras/test/e2e/internal/utils"
)

var _ = Describe("ORAS beginners:", func() {
	When("running pull command", func() {
		It("should show help description with feature flags", func() {
			out := ORAS("pull", "--help").MatchKeyWords(ExampleDesc).Exec().Out
			gomega.Expect(out).Should(gbytes.Say("--include-subject\\s+%s", regexp.QuoteMeta(feature.Preview.Mark)))
		})
	})
})

var _ = Describe("Remote registry users:", func() {
	When("pulling images from remote registry", func() {
		var (
			configName = "test.config"
		)

		It("should pull all files in an image to a target folder", func() {
			pullRoot := "pulled"
			tempDir := PrepareTempFiles()
			stateKeys := append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageConfigStateKey(configName))
			ORAS("pull", RegistryRef(Host, ImageRepo, foobar.Tag), "-v", "--config", configName, "-o", pullRoot).
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

			ORAS("pull", RegistryRef(Host, ImageRepo, foobar.Tag), "-v", "-o", pullRoot, "--keep-old-files").
				ExpectFailure().
				WithDescription("fail if overwrite old files are disabled")
		})

		It("should skip config if media type not matching", func() {
			pullRoot := "pulled"
			tempDir := PrepareTempFiles()
			stateKeys := append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageConfigStateKey(oras.MediaTypeUnknownConfig))
			ORAS("pull", RegistryRef(Host, ImageRepo, foobar.Tag), "-v", "--config", fmt.Sprintf("%s:%s", configName, "???"), "-o", pullRoot).
				MatchStatus(stateKeys, true, len(stateKeys)).
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
			ORAS("pull", RegistryRef(Host, ImageRepo, "multi"), "--platform", "linux/amd64", "-v", "-o", GinkgoT().TempDir()).
				MatchStatus(multi_arch.LinuxAMD64StateKeys, true, len(multi_arch.LinuxAMD64StateKeys)).Exec()
		})
	})
})

var _ = Describe("OCI image layout users:", func() {
	When("pulling images", func() {
		var (
			configName = "test.config"
		)
		prepare := func(root string, repo string, tagOrDigest string) {
			ORAS("cp", RegistryRef(Host, repo, tagOrDigest), Flags.ToLayout, LayoutRef(root, tagOrDigest), "-r").WithDescription("prepare oci layout test env").Exec()
		}
		It("should pull all files in an image to a target folder", func() {
			pullRoot := "pulled"
			tempDir := PrepareTempFiles()
			prepare(tempDir, ArtifactRepo, foobar.Tag)
			stateKeys := append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageConfigStateKey(configName))
			ORAS("pull", Flags.Layout, LayoutRef(tempDir, foobar.Tag), "-v", "--config", configName, "-o", pullRoot).
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

			ORAS("pull", Flags.Layout, LayoutRef(tempDir, foobar.Tag), "-v", "-o", pullRoot, "--keep-old-files").
				ExpectFailure().
				WithDescription("fail if overwrite old files are disabled")
		})

		It("should skip config if media type does not match", func() {
			pullRoot := "pulled"
			tempDir := PrepareTempFiles()
			prepare(tempDir, ArtifactRepo, foobar.Tag)
			stateKeys := append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageConfigStateKey(oras.MediaTypeUnknownConfig))
			ORAS("pull", Flags.Layout, LayoutRef(tempDir, foobar.Tag), "-v", "--config", fmt.Sprintf("%s:%s", configName, "???"), "-o", pullRoot).
				MatchStatus(stateKeys, true, len(stateKeys)).
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

		It("should pull subject", func() {
			tempDir := GinkgoT().TempDir()
			prepare(tempDir, ArtifactRepo, foobar.Tag)
			stateKeys := append(append(
				foobar.ImageLayerStateKeys,
				foobar.ManifestStateKey),
				foobar.ImageReferrersStateKeys...,
			)
			ORAS("pull", Flags.Layout, LayoutRef(tempDir, foobar.SignatureImageReferrer.Digest.String()), "-v", "--include-subject").
				MatchStatus(stateKeys, true, len(stateKeys)).
				WithWorkDir(tempDir).Exec()
		})

		It("should pull specific platform", func() {
			tempDir := GinkgoT().TempDir()
			prepare(tempDir, ImageRepo, multi_arch.Tag)
			ORAS("pull", Flags.Layout, LayoutRef(tempDir, multi_arch.Tag), "--platform", "linux/amd64", "-v", "-o", tempDir).
				MatchStatus(multi_arch.LinuxAMD64StateKeys, true, len(multi_arch.LinuxAMD64StateKeys)).Exec()
		})
	})
})
