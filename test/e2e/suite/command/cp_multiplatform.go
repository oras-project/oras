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
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	ma "oras.land/oras/test/e2e/internal/testdata/multi_arch"
	. "oras.land/oras/test/e2e/internal/utils"
)

func cpMultiPlatformTestRepo(text string) string {
	return fmt.Sprintf("command/copy/multiplatform/%d/%s", GinkgoRandomSeed(), text)
}

var _ = Describe("Multi-platform copy users:", func() {
	When("running `cp` with multiple platforms", func() {
		It("should copy multiple platforms of image to a new repository", func() {
			src := RegistryRef(ZOTHost, ImageRepo, ma.Tag)
			dst := RegistryRef(ZOTHost, cpMultiPlatformTestRepo("multi-platform"), "copiedMulti")

			// Copy multiple platforms: linux/amd64 and linux/arm64
			ORAS("cp", src, dst, "--platform", "linux/amd64,linux/arm64").
				MatchKeyWords("linux/amd64", "linux/arm64").
				Exec()

			// validate
			// Check that the resulting manifest index contains only the selected platforms
			manifest := ORAS("manifest", "fetch", dst).Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(manifest, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(2)) // Should have 2 manifests for the 2 selected platforms
			platformsFound := make(map[string]bool)
			for _, manifest := range index.Manifests {
				if manifest.Platform != nil {
					platformsFound[fmt.Sprintf("%s/%s", manifest.Platform.OS, manifest.Platform.Architecture)] = true
				}
			}
			Expect(platformsFound["linux/amd64"]).To(BeTrue())
			Expect(platformsFound["linux/arm64"]).To(BeTrue())
		})

		It("should fail to copy multiple platforms when some platforms are not available", func() {
			src := RegistryRef(ZOTHost, ImageRepo, ma.Tag)
			dst := RegistryRef(ZOTHost, cpMultiPlatformTestRepo("missing-platform"), "copiedMissing")

			// Attempt to copy platforms that include one that doesn't exist
			ORAS("cp", src, dst, "--platform", "linux/amd64,linux/nonexistent").
				ExpectFailure().
				MatchErrKeyWords("only 1 of 2 requested platforms were matched", "unmatched platforms: [linux/nonexistent]").
				Exec()
		})

		It("should copy multiple platforms of image with recursive flag", func() {
			src := RegistryRef(ZOTHost, ArtifactRepo, ma.Tag)
			dstRepo := cpMultiPlatformTestRepo("multi-platform-recursive")
			dst := RegistryRef(ZOTHost, dstRepo, "copiedMultiRecursive")

			// Copy multiple platforms with referrers: linux/amd64 and linux/arm64
			ORAS("cp", src, dst, "-r", "--platform", "linux/amd64,linux/arm64").
				Exec()

			// validate
			// Check that the resulting manifest index contains only the selected platforms
			manifest := ORAS("manifest", "fetch", dst).Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(manifest, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(2)) // Should have 2 manifests for the 2 selected platforms
			platformsFound := make(map[string]bool)
			for _, manifest := range index.Manifests {
				if manifest.Platform != nil {
					platformsFound[fmt.Sprintf("%s/%s", manifest.Platform.OS, manifest.Platform.Architecture)] = true
				}
			}
			Expect(platformsFound["linux/amd64"]).To(BeTrue())
			Expect(platformsFound["linux/arm64"]).To(BeTrue())

			// Also check that referrers were copied for the selected platforms
			ORAS("discover", dst, "--artifact-type", "signature").Exec()
		})

		It("should copy a single platform when only one platform is specified", func() {
			src := RegistryRef(ZOTHost, ImageRepo, ma.Tag)
			dst := RegistryRef(ZOTHost, cpMultiPlatformTestRepo("single-platform"), "copiedSingle")

			// Copy a single platform: linux/amd64
			ORAS("cp", src, dst, "--platform", "linux/amd64").
				Exec()

			// validate
			manifest := ORAS("manifest", "fetch", dst).Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(manifest, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1)) // Should have 1 manifest for the selected platform
			Expect(index.Manifests[0].Platform).ToNot(BeNil())
			Expect(index.Manifests[0].Platform.OS).To(Equal("linux"))
			Expect(index.Manifests[0].Platform.Architecture).To(Equal("amd64"))
		})

		It("should copy multiple platforms with complex platform strings including variants", func() {
			src := RegistryRef(ZOTHost, ImageRepo, ma.Tag)
			dst := RegistryRef(ZOTHost, cpMultiPlatformTestRepo("complex-platform"), "copiedComplex")

			// Copy multiple platforms: linux/amd64 and linux/arm/v7 (with variant)
			ORAS("cp", src, dst, "--platform", "linux/amd64,linux/arm/v7").
				Exec()

			// validate
			manifest := ORAS("manifest", "fetch", dst).Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(manifest, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(2)) // Should have 2 manifests for the 2 selected platforms
			platformsFound := make(map[string]bool)
			for _, manifest := range index.Manifests {
				if manifest.Platform != nil {
					platformStr := fmt.Sprintf("%s/%s", manifest.Platform.OS, manifest.Platform.Architecture)
					if manifest.Platform.Variant != "" {
						platformStr += "/" + manifest.Platform.Variant
					}
					platformsFound[platformStr] = true
				}
			}
			Expect(platformsFound["linux/amd64"]).To(BeTrue())
			Expect(platformsFound["linux/arm/v7"]).To(BeTrue())
		})
	})
})
