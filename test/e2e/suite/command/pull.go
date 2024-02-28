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
	"oras.land/oras/test/e2e/internal/testdata/artifact/blob"
	"oras.land/oras/test/e2e/internal/testdata/artifact/config"
	"oras.land/oras/test/e2e/internal/testdata/artifact/empty"
	"oras.land/oras/test/e2e/internal/testdata/artifact/unnamed"
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

		hintMsg := func(reference string) string {
			return fmt.Sprintf("Skipped pulling layers without file name in \"org.opencontainers.image.title\"\nUse 'oras copy %s --to-oci-layout <layout-dir>' to pull all layers.\n", reference)
		}
		It("should show hint for unnamed layer", func() {
			tempDir := PrepareTempFiles()
			ref := RegistryRef(ZOTHost, ArtifactRepo, unnamed.Tag)
			ORAS("pull", ref).
				WithWorkDir(tempDir).
				MatchContent(hintMsg(ref)).Exec()
		})

		It("should not show hint for unnamed config blob", func() {
			tempDir := PrepareTempFiles()
			ref := RegistryRef(ZOTHost, ImageRepo, foobar.Tag)
			out := ORAS("pull", ref).WithWorkDir(tempDir).Exec().Out
			gomega.Expect(out).ShouldNot(gbytes.Say(hintMsg(ref)))
		})

		It("should not show hint for empty layer", func() {
			tempDir := PrepareTempFiles()
			ref := RegistryRef(ZOTHost, ArtifactRepo, empty.Tag)
			out := ORAS("pull", ref).WithWorkDir(tempDir).Exec().Out
			gomega.Expect(out).ShouldNot(gbytes.Say(hintMsg(ref)))
		})

		It("should fail and show detailed error description if no argument provided", func() {
			err := ORAS("pull").ExpectFailure().Exec().Err
			gomega.Expect(err).Should(gbytes.Say("Error"))
			gomega.Expect(err).Should(gbytes.Say("\nUsage: oras pull"))
			gomega.Expect(err).Should(gbytes.Say("\n"))
			gomega.Expect(err).Should(gbytes.Say(`Run "oras pull -h"`))
		})

		It("should fail if password is wrong with registry error prefix", func() {
			ORAS("pull", RegistryRef(ZOTHost, ArtifactRepo, empty.Tag), "-u", Username, "-p", "???").
				MatchErrKeyWords(RegistryErrorPrefix).ExpectFailure().Exec()
		})

		It("should fail if artifact is not found with registry error prefix", func() {
			ORAS("pull", RegistryRef(ZOTHost, ArtifactRepo, InvalidTag)).
				MatchErrKeyWords(RegistryErrorPrefix).ExpectFailure().Exec()
		})

		It("should fail if artifact is not found from OCI layout", func() {
			root := PrepareTempOCI(ArtifactRepo)
			ORAS("pull", Flags.Layout, LayoutRef(root, InvalidTag)).
				MatchErrKeyWords("Error: ").ExpectFailure().Exec()
		})
	})
})

var _ = Describe("OCI spec 1.1 registry users:", func() {
	When("pulling images from remote registry", func() {
		var (
			configName = "test.config"
		)

		It("should pull all files in an image to a target folder", func() {
			pullRoot := "pulled"
			tempDir := PrepareTempFiles()
			stateKeys := append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageConfigStateKey(configName))
			ORAS("pull", RegistryRef(ZOTHost, ImageRepo, foobar.Tag), "-v", "--config", configName, "-o", pullRoot).
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

			ORAS("pull", RegistryRef(ZOTHost, ImageRepo, foobar.Tag), "-v", "-o", pullRoot, "--keep-old-files").
				ExpectFailure().
				WithDescription("fail if overwrite old files are disabled")
		})

		It("should skip config if media type not matching", func() {
			pullRoot := "pulled"
			tempDir := PrepareTempFiles()
			stateKeys := append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey)
			ORAS("pull", RegistryRef(ZOTHost, ImageRepo, foobar.Tag), "-v", "--config", fmt.Sprintf("%s:%s", configName, "???"), "-o", pullRoot).
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
			stateKeys := append(append(
				foobar.ImageLayerStateKeys,
				foobar.ManifestStateKey),
				foobar.ImageReferrersStateKeys...,
			)
			ORAS("pull", RegistryRef(ZOTHost, ArtifactRepo, foobar.SignatureImageReferrer.Digest.String()), "-v", "--include-subject").
				MatchStatus(stateKeys, true, len(stateKeys)).
				WithWorkDir(tempDir).Exec()
		})

		It("should pull specific platform", func() {
			ORAS("pull", RegistryRef(ZOTHost, ImageRepo, "multi"), "--platform", "linux/amd64", "-v", "-o", GinkgoT().TempDir()).
				MatchStatus(multi_arch.LinuxAMD64StateKeys, true, len(multi_arch.LinuxAMD64StateKeys)).Exec()
		})

		It("should pull an artifact with blob", func() {
			pullRoot := GinkgoT().TempDir()
			ORAS("pull", RegistryRef(ZOTHost, ArtifactRepo, blob.Tag), "-v", "-o", pullRoot).Exec()
			Expect(filepath.Join(pullRoot, multi_arch.LayerName)).Should(BeAnExistingFile())
		})

		It("should pull an artifact with config", func() {
			pullRoot := GinkgoT().TempDir()
			ORAS("pull", RegistryRef(ZOTHost, ArtifactRepo, config.Tag), "-v", "-o", pullRoot).Exec()
			Expect(filepath.Join(pullRoot, multi_arch.LayerName)).Should(BeAnExistingFile())
		})

		It("should copy an artifact with blob", func() {
			repo := cpTestRepo("artifact-with-blob")
			stateKeys := append(append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageReferrerConfigStateKeys[0]), foobar.ImageReferrersStateKeys...)
			src := RegistryRef(ZOTHost, ArtifactRepo, foobar.SignatureImageReferrer.Digest.String())
			dst := RegistryRef(FallbackHost, repo, "")
			ORAS("cp", "-r", src, dst, "-v").MatchStatus(stateKeys, true, len(stateKeys)).Exec()
			CompareRef(src, RegistryRef(FallbackHost, repo, foobar.SignatureImageReferrer.Digest.String()))
			ORAS("discover", "-o", "tree", RegistryRef(FallbackHost, repo, foobar.Digest)).
				WithDescription("discover referrer via subject").MatchKeyWords(foobar.SignatureImageReferrer.Digest.String(), foobar.SBOMImageReferrer.Digest.String()).Exec()
		})
	})
})

var _ = Describe("OCI spec 1.0 registry users:", func() {
	It("should pull all files in an image to a target folder", func() {
		pullRoot := "pulled"
		configName := "test.config"
		tempDir := PrepareTempFiles()
		stateKeys := append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageConfigStateKey(configName))
		ORAS("pull", RegistryRef(FallbackHost, ArtifactRepo, foobar.Tag), "-v", "--config", configName, "-o", pullRoot).
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

		ORAS("pull", RegistryRef(FallbackHost, ArtifactRepo, foobar.Tag), "-v", "-o", pullRoot, "--keep-old-files").
			ExpectFailure().
			WithDescription("fail if overwrite old files are disabled")
	})

	It("should pull subject", func() {
		tempDir := GinkgoT().TempDir()
		stateKeys := append(append(
			foobar.ImageLayerStateKeys,
			foobar.ManifestStateKey),
			foobar.ImageReferrersStateKeys...,
		)
		ORAS("pull", RegistryRef(FallbackHost, ArtifactRepo, foobar.SignatureImageReferrer.Digest.String()), "-v", "--include-subject").
			MatchStatus(stateKeys, true, len(stateKeys)).
			WithWorkDir(tempDir).Exec()
	})
})

var _ = Describe("OCI image layout users:", func() {
	When("pulling images", func() {
		var (
			configName = "test.config"
		)
		It("should pull all files in an image to a target folder", func() {
			pullRoot := "pulled"
			root := PrepareTempOCI(ImageRepo)
			stateKeys := append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageConfigStateKey(configName))
			ORAS("pull", Flags.Layout, LayoutRef(root, foobar.Tag), "-v", "--config", configName, "-o", pullRoot).
				MatchStatus(stateKeys, true, len(stateKeys)).
				WithWorkDir(root).Exec()
			// check config
			configPath := filepath.Join(root, pullRoot, configName)
			Expect(configPath).Should(BeAnExistingFile())
			f, err := os.Open(configPath)
			Expect(err).ShouldNot(HaveOccurred())
			defer f.Close()
			Eventually(gbytes.BufferReader(f)).Should(gbytes.Say("{}"))
			for _, f := range foobar.ImageLayerNames {
				// check layers
				Binary("diff", filepath.Join(root, "foobar", f), filepath.Join(pullRoot, f)).
					WithWorkDir(root).Exec()
			}

			ORAS("pull", Flags.Layout, LayoutRef(root, foobar.Tag), "-v", "-o", pullRoot, "--keep-old-files").
				ExpectFailure().
				WithDescription("fail if overwrite old files are disabled")
		})

		It("should skip config if media type does not match", func() {
			pullRoot := "pulled"
			root := PrepareTempOCI(ImageRepo)
			stateKeys := append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey)
			ORAS("pull", Flags.Layout, LayoutRef(root, foobar.Tag), "-v", "--config", fmt.Sprintf("%s:%s", configName, "???"), "-o", pullRoot).
				MatchStatus(stateKeys, true, len(stateKeys)).
				WithWorkDir(root).Exec()
			// check config
			Expect(filepath.Join(pullRoot, configName)).ShouldNot(BeAnExistingFile())
			for _, f := range foobar.ImageLayerNames {
				// check layers
				Binary("diff", filepath.Join(root, "foobar", f), filepath.Join(pullRoot, f)).
					WithWorkDir(root).
					WithDescription("should download identical file " + f).Exec()
			}
		})

		It("should pull subject", func() {
			root := PrepareTempOCI(ArtifactRepo)
			stateKeys := append(append(
				foobar.ImageLayerStateKeys,
				foobar.ManifestStateKey),
				foobar.ImageReferrersStateKeys...,
			)
			ORAS("pull", Flags.Layout, LayoutRef(root, foobar.SignatureImageReferrer.Digest.String()), "-v", "--include-subject").
				MatchStatus(stateKeys, true, len(stateKeys)).
				WithWorkDir(root).Exec()
		})

		It("should pull specific platform", func() {
			root := PrepareTempOCI(ImageRepo)
			ORAS("pull", Flags.Layout, LayoutRef(root, multi_arch.Tag), "--platform", "linux/amd64", "-v", "-o", root).
				MatchStatus(multi_arch.LinuxAMD64StateKeys, true, len(multi_arch.LinuxAMD64StateKeys)).Exec()
		})
	})
})

var _ = Describe("OCI image spec v1.1.0-rc2 artifact users:", func() {
	It("should pull all files in an image to a target folder", func() {
		pullRoot := "pulled"
		configName := "test.config"
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

	It("should pull subject", func() {
		tempDir := GinkgoT().TempDir()
		stateKeys := append(append(
			foobar.ImageLayerStateKeys,
			foobar.ManifestStateKey),
			foobar.ArtifactReferrerStateKeys...,
		)
		ORAS("pull", RegistryRef(Host, ArtifactRepo, foobar.SignatureArtifactReferrer.Digest.String()), "-v", "--include-subject").
			MatchStatus(stateKeys, true, len(stateKeys)).
			WithWorkDir(tempDir).Exec()
	})
})
