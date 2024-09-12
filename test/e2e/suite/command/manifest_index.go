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
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/test/e2e/internal/testdata/multi_arch"
	"oras.land/oras/test/e2e/internal/testdata/nonjson_config"
	. "oras.land/oras/test/e2e/internal/utils"
)

var _ = Describe("ORAS beginners:", func() {
	When("running manifest index command", func() {
		When("running `manifest index create`", func() {
			It("should show help doc with alias", func() {
				ORAS("manifest", "index", "create", "--help").MatchKeyWords("Aliases", "pack").Exec()
			})
		})
	})
})

func indexTestRepo(subcommand string, text string) string {
	return fmt.Sprintf("command/index/%d/%s/%s", GinkgoRandomSeed(), subcommand, text)
}

func ValidateIndex(content []byte, manifests []ocispec.Descriptor) {
	var index ocispec.Index
	Expect(json.Unmarshal(content, &index)).ShouldNot(HaveOccurred())
	Expect(index.Manifests).To(Equal(manifests))
}

var _ = Describe("1.1 registry users:", func() {
	When("running `manifest index create`", func() {
		It("should create index by using source manifest digests", func() {
			testRepo := indexTestRepo("create", "by-digest")
			CopyZOTRepo(ImageRepo, testRepo)
			ORAS("manifest", "index", "create", RegistryRef(ZOTHost, testRepo, "latest"),
				string(multi_arch.LinuxAMD64.Digest), string(multi_arch.LinuxARM64.Digest)).
				MatchKeyWords("Fetched", "sha256:9d84a5716c66a1d1b9c13f8ed157ba7d1edfe7f9b8766728b8a1f25c0d9c14c1",
					"Fetched", "sha256:4f93460061882467e6fb3b772dc6ab72130d9ac1906aed2fc7589a5cd145433c",
					"Pushed", "sha256:cce9590b1193d8bcb70467e2381dc81e77869be4801c09abe9bc274b6a1d2001").Exec()
			// verify
			content := ORAS("manifest", "fetch", RegistryRef(ZOTHost, testRepo, "latest")).Exec().Out.Contents()
			expectedManifests := []ocispec.Descriptor{multi_arch.LinuxAMD64, multi_arch.LinuxARM64}
			ValidateIndex(content, expectedManifests)
		})

		It("should create index by using source manifest tags", func() {
			testRepo := indexTestRepo("create", "by-tag")
			CopyZOTRepo(ImageRepo, testRepo)
			ORAS("manifest", "index", "create", RegistryRef(ZOTHost, testRepo, "latest"),
				"linux-arm64", "linux-amd64").
				MatchKeyWords("Fetched", "linux-arm64",
					"Fetched", "linux-amd64",
					"Pushed", "sha256:5c98cfc90e390c575679370a5dc5e37b52e854bbb7b9cb80cc1f30b56b8d183e").Exec()
			// verify
			content := ORAS("manifest", "fetch", RegistryRef(ZOTHost, testRepo, "latest")).Exec().Out.Contents()
			expectedManifests := []ocispec.Descriptor{multi_arch.LinuxARM64, multi_arch.LinuxAMD64}
			ValidateIndex(content, expectedManifests)
		})

		It("should create index without tagging it", func() {
			testRepo := indexTestRepo("create", "no-tag")
			CopyZOTRepo(ImageRepo, testRepo)
			ORAS("manifest", "index", "create", RegistryRef(ZOTHost, testRepo, ""),
				"linux-arm64", "linux-amd64", "sha256:58efe73e78fe043ca31b89007a025c594ce12aa7e6da27d21c7b14b50112e255").
				MatchKeyWords("Pushed", "sha256:820503ae4fecfdb841b5b6acc8718c8c5b298cf6b8f2259010f370052341cec8").Exec()
			// verify
			content := ORAS("manifest", "fetch", RegistryRef(ZOTHost, testRepo, "sha256:820503ae4fecfdb841b5b6acc8718c8c5b298cf6b8f2259010f370052341cec8")).
				Exec().Out.Contents()
			expectedManifests := []ocispec.Descriptor{multi_arch.LinuxARM64, multi_arch.LinuxAMD64, multi_arch.LinuxARMV7}
			ValidateIndex(content, expectedManifests)
		})

		It("should create index with multiple tags", func() {
			testRepo := indexTestRepo("create", "multiple-tags")
			CopyZOTRepo(ImageRepo, testRepo)
			ORAS("manifest", "index", "create", fmt.Sprintf("%s,t1,t2,t3", RegistryRef(ZOTHost, testRepo, "t0")),
				"sha256:58efe73e78fe043ca31b89007a025c594ce12aa7e6da27d21c7b14b50112e255", "linux-arm64", "linux-amd64").
				MatchKeyWords("Fetched", "Pushed", "Tagged",
					"sha256:bfa1728d6292d5fa7689f8f4daa145ee6f067b5779528c6e059d1132745ef508").Exec()
			// verify
			expectedManifests := []ocispec.Descriptor{multi_arch.LinuxARMV7, multi_arch.LinuxARM64, multi_arch.LinuxAMD64}
			content := ORAS("manifest", "fetch", RegistryRef(ZOTHost, testRepo, "t0")).Exec().Out.Contents()
			ValidateIndex(content, expectedManifests)
			content = ORAS("manifest", "fetch", RegistryRef(ZOTHost, testRepo, "t1")).Exec().Out.Contents()
			ValidateIndex(content, expectedManifests)
			content = ORAS("manifest", "fetch", RegistryRef(ZOTHost, testRepo, "t2")).Exec().Out.Contents()
			ValidateIndex(content, expectedManifests)
			content = ORAS("manifest", "fetch", RegistryRef(ZOTHost, testRepo, "t3")).Exec().Out.Contents()
			ValidateIndex(content, expectedManifests)
		})

		It("should create nested indexes", func() {
			testRepo := indexTestRepo("create", "nested-index")
			CopyZOTRepo(ImageRepo, testRepo)
			ORAS("manifest", "index", "create", RegistryRef(ZOTHost, testRepo, "nested"), "multi").Exec()
			// verify
			content := ORAS("manifest", "fetch", RegistryRef(ZOTHost, testRepo, "nested")).Exec().Out.Contents()
			expectedManifests := []ocispec.Descriptor{multi_arch.DescriptorObject}
			ValidateIndex(content, expectedManifests)
		})

		It("should create index from image with non-json config", func() {
			testRepo := indexTestRepo("create", "nonjson-config")
			CopyZOTRepo(ImageRepo, testRepo)
			ORAS("manifest", "index", "create", RegistryRef(ZOTHost, testRepo, "unusual-config"),
				"nonjson-config").Exec()
			// verify
			content := ORAS("manifest", "fetch", RegistryRef(ZOTHost, testRepo, "unusual-config")).Exec().Out.Contents()
			expectedManifests := []ocispec.Descriptor{nonjson_config.Descriptor}
			ValidateIndex(content, expectedManifests)
		})

		It("should output created index to file", func() {
			testRepo := indexTestRepo("create", "output-to-file")
			CopyZOTRepo(ImageRepo, testRepo)
			filePath := filepath.Join(GinkgoT().TempDir(), "createdIndex")
			ORAS("manifest", "index", "create", RegistryRef(ZOTHost, testRepo, ""), string(multi_arch.LinuxAMD64.Digest), "--output", filePath).Exec()
			MatchFile(filePath, multi_arch.CreatedIndex, DefaultTimeout)
		})

		It("should output created index to stdout", func() {
			testRepo := indexTestRepo("create", "output-to-stdout")
			CopyZOTRepo(ImageRepo, testRepo)
			ORAS("manifest", "index", "create", RegistryRef(ZOTHost, testRepo, ""), string(multi_arch.LinuxAMD64.Digest),
				"--output", "-").MatchKeyWords(multi_arch.CreatedIndex).Exec()
		})

		It("should fail if given a reference that does not exist in the repo", func() {
			testRepo := indexTestRepo("create", "nonexist-ref")
			CopyZOTRepo(ImageRepo, testRepo)
			ORAS("manifest", "index", "create", RegistryRef(ZOTHost, testRepo, ""),
				"does-not-exist").ExpectFailure().
				MatchErrKeyWords("Error", "could not find", "does-not-exist").Exec()
		})
	})
})

var _ = Describe("OCI image layout users:", func() {
	When("running `manifest index create`", func() {
		It("should create an index with source manifest digest", func() {
			root := PrepareTempOCI(ImageRepo)
			indexRef := LayoutRef(root, "latest")
			ORAS("manifest", "index", "create", Flags.Layout, indexRef, string(multi_arch.LinuxAMD64.Digest)).
				WithWorkDir(root).Exec()
			// verify
			content := ORAS("manifest", "fetch", Flags.Layout, indexRef).Exec().Out.Contents()
			expectedManifests := []ocispec.Descriptor{multi_arch.LinuxAMD64}
			ValidateIndex(content, expectedManifests)
		})

		It("should create an index with source manifest tag", func() {
			root := PrepareTempOCI(ImageRepo)
			indexRef := LayoutRef(root, "latest")
			ORAS("manifest", "index", "create", Flags.Layout, indexRef, "linux-amd64").
				WithWorkDir(root).Exec()
			// verify
			content := ORAS("manifest", "fetch", Flags.Layout, indexRef).Exec().Out.Contents()
			expectedManifests := []ocispec.Descriptor{multi_arch.LinuxAMD64}
			ValidateIndex(content, expectedManifests)
		})

		It("should create an index without tagging it", func() {
			root := PrepareTempOCI(ImageRepo)
			indexRef := LayoutRef(root, "")
			ORAS("manifest", "index", "create", Flags.Layout, indexRef, "linux-amd64").
				WithWorkDir(root).MatchKeyWords("Digest: sha256:c543059818cb70e6442597a33454ec1e3d3a2bdb526c17875578d33c2ddcf72e").Exec()
			// verify
			content := ORAS("manifest", "fetch", Flags.Layout, LayoutRef(root, "sha256:c543059818cb70e6442597a33454ec1e3d3a2bdb526c17875578d33c2ddcf72e")).Exec().Out.Contents()
			expectedManifests := []ocispec.Descriptor{multi_arch.LinuxAMD64}
			ValidateIndex(content, expectedManifests)
		})

		It("should create an index with multiple tags", func() {
			root := PrepareTempOCI(ImageRepo)
			indexRef := fmt.Sprintf("%s,t1,t2,t3", LayoutRef(root, "t0"))
			ORAS("manifest", "index", "create", Flags.Layout, indexRef, "linux-amd64").WithWorkDir(root).Exec()
			// verify
			expectedManifests := []ocispec.Descriptor{multi_arch.LinuxAMD64}
			content := ORAS("manifest", "fetch", Flags.Layout, LayoutRef(root, "t0")).Exec().Out.Contents()
			ValidateIndex(content, expectedManifests)
			content = ORAS("manifest", "fetch", Flags.Layout, LayoutRef(root, "t1")).Exec().Out.Contents()
			ValidateIndex(content, expectedManifests)
			content = ORAS("manifest", "fetch", Flags.Layout, LayoutRef(root, "t2")).Exec().Out.Contents()
			ValidateIndex(content, expectedManifests)
			content = ORAS("manifest", "fetch", Flags.Layout, LayoutRef(root, "t3")).Exec().Out.Contents()
			ValidateIndex(content, expectedManifests)
		})

		It("should create nested indexes", func() {
			root := PrepareTempOCI(ImageRepo)
			indexRef := LayoutRef(root, "nested-index")
			ORAS("manifest", "index", "create", Flags.Layout, indexRef, "multi").WithWorkDir(root).Exec()
			// verify
			content := ORAS("manifest", "fetch", Flags.Layout, indexRef).Exec().Out.Contents()
			expectedManifests := []ocispec.Descriptor{multi_arch.DescriptorObject}
			ValidateIndex(content, expectedManifests)
		})

		It("should create index from image with non-json config", func() {
			root := PrepareTempOCI(ImageRepo)
			indexRef := LayoutRef(root, "unusual-config")
			ORAS("manifest", "index", "create", Flags.Layout, indexRef, "nonjson-config").WithWorkDir(root).Exec()
			// verify
			content := ORAS("manifest", "fetch", Flags.Layout, indexRef).Exec().Out.Contents()
			expectedManifests := []ocispec.Descriptor{nonjson_config.Descriptor}
			ValidateIndex(content, expectedManifests)
		})

		It("should output created index to file", func() {
			root := PrepareTempOCI(ImageRepo)
			indexRef := LayoutRef(root, "output-to-file")
			filePath := filepath.Join(GinkgoT().TempDir(), "createdIndex")
			ORAS("manifest", "index", "create", Flags.Layout, indexRef, string(multi_arch.LinuxAMD64.Digest), "--output", filePath).Exec()
			MatchFile(filePath, multi_arch.CreatedIndex, DefaultTimeout)
		})

		It("should output created index to stdout", func() {
			root := PrepareTempOCI(ImageRepo)
			indexRef := LayoutRef(root, "output-to-stdout")
			ORAS("manifest", "index", "create", Flags.Layout, indexRef, string(multi_arch.LinuxAMD64.Digest),
				"--output", "-").MatchKeyWords(multi_arch.CreatedIndex).Exec()
		})

		It("should fail if given a reference that does not exist in the repo", func() {
			root := PrepareTempOCI(ImageRepo)
			indexRef := LayoutRef(root, "latest")
			ORAS("manifest", "index", "create", Flags.Layout, indexRef, "does-not-exist").ExpectFailure().
				MatchErrKeyWords("Error", "could not find", "does-not-exist").Exec()
		})

		It("should fail if given a digest that is not a manifest", func() {
			root := PrepareTempOCI(ImageRepo)
			indexRef := LayoutRef(root, "latest")
			ORAS("manifest", "index", "create", Flags.Layout, indexRef, "sha256:02c15a8d1735c65bb8ca86c716615d3c0d8beb87dc68ed88bb49192f90b184e2").ExpectFailure().
				MatchErrKeyWords("is not a manifest").Exec()
		})
	})
})
