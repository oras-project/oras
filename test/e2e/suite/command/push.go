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
	"regexp"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/test/e2e/internal/testdata/feature"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	. "oras.land/oras/test/e2e/internal/utils"
	"oras.land/oras/test/e2e/internal/utils/match"
)

var _ = Describe("ORAS beginners:", func() {
	When("running push command", func() {
		It("should show help description with feature flags", func() {
			out := ORAS("push", "--help").MatchKeyWords(ExampleDesc).Exec().Out
			gomega.Expect(out).Should(gbytes.Say("--image-spec string\\s+%s", regexp.QuoteMeta(feature.Experimental.Mark)))
		})
	})
})

var _ = Describe("Remote registry users:", func() {
	tag := "e2e"
	When("pushing to registy without OCI artifact support", func() {
		repoPrefix := fmt.Sprintf("command/push/%d", GinkgoRandomSeed())
		statusKeys := []match.StateKey{
			foobar.ImageConfigStateKey("application/vnd.unknown.config.v1+json"),
			foobar.FileBarStateKey,
		}
		It("should push files without customized media types", func() {
			repo := fmt.Sprintf("%s/%s", repoPrefix, "no-mediatype")
			tempDir := PrepareTempFiles()
			ref := RegistryRef(Host, repo, tag)

			ORAS("push", ref, foobar.FileBarName, "-v").
				MatchStatus(statusKeys, true, len(statusKeys)).
				WithWorkDir(tempDir).Exec()

			// validate
			fetched := ORAS("manifest", "fetch", ref).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Layers).Should(ContainElements(foobar.BlobBarDescriptor("application/vnd.oci.image.layer.v1.tar")))
		})

		It("should push files with path validation disabled", func() {
			repo := fmt.Sprintf("%s/%s", repoPrefix, "disable-path-validation")
			ref := RegistryRef(Host, repo, tag)
			absBarName := filepath.Join(PrepareTempFiles(), foobar.FileBarName)

			ORAS("push", ref, absBarName, "-v", "--disable-path-validation").
				Exec()

			// validate
			fetched := ORAS("manifest", "fetch", ref).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Layers).Should(ContainElements(ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.layer.v1.tar",
				Digest:    digest.Digest(foobar.BarBlobDigest),
				Size:      3,
				Annotations: map[string]string{
					"org.opencontainers.image.title": absBarName,
				},
			}))
		})

		It("should fail path validation when pushing file with absolute path", func() {
			repo := fmt.Sprintf("%s/%s", repoPrefix, "path-validation")
			ref := RegistryRef(Host, repo, tag)
			absBarName := filepath.Join(PrepareTempFiles(), foobar.FileBarName)
			// test
			ORAS("push", ref, absBarName, "-v").
				MatchErrKeyWords("--disable-path-validation").
				ExpectFailure().
				Exec()
		})

		It("should push files and tag", func() {
			repo := fmt.Sprintf("%s/%s", repoPrefix, "multi-tag")
			tempDir := PrepareTempFiles()
			extraTag := "2e2"

			ORAS("push", fmt.Sprintf("%s,%s", RegistryRef(Host, repo, tag), extraTag), foobar.FileBarName, "-v").
				MatchStatus(statusKeys, true, len(statusKeys)).
				WithWorkDir(tempDir).Exec()

			// validate
			fetched := ORAS("manifest", "fetch", RegistryRef(Host, repo, tag)).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Layers).Should(ContainElements(foobar.BlobBarDescriptor("application/vnd.oci.image.layer.v1.tar")))

			fetched = ORAS("manifest", "fetch", RegistryRef(Host, repo, extraTag)).Exec().Out.Contents()
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Layers).Should(ContainElements(foobar.BlobBarDescriptor("application/vnd.oci.image.layer.v1.tar")))
		})

		It("should push files with customized media types", func() {
			repo := fmt.Sprintf("%s/%s", repoPrefix, "layer-mediatype")
			layerType := "layer.type"
			tempDir := PrepareTempFiles()
			ORAS("push", RegistryRef(Host, repo, tag), foobar.FileBarName+":"+layerType, "-v").
				MatchStatus(statusKeys, true, len(statusKeys)).
				WithWorkDir(tempDir).Exec()
			// validate
			fetched := ORAS("manifest", "fetch", RegistryRef(Host, repo, tag)).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Layers).Should(ContainElements(foobar.BlobBarDescriptor(layerType)))
		})

		It("should push files with manifest exported", func() {
			repo := fmt.Sprintf("%s/%s", repoPrefix, "export-manifest")
			layerType := "layer.type"
			tempDir := PrepareTempFiles()
			exportPath := "packed.json"
			ORAS("push", RegistryRef(Host, repo, tag), foobar.FileBarName+":"+layerType, "-v", "--export-manifest", exportPath).
				MatchStatus(statusKeys, true, len(statusKeys)).
				WithWorkDir(tempDir).Exec()
			// validate
			fetched := ORAS("manifest", "fetch", RegistryRef(Host, repo, tag)).Exec().Out.Contents()
			MatchFile(filepath.Join(tempDir, exportPath), string(fetched), DefaultTimeout)
		})

		It("should push files with customized config file", func() {
			repo := fmt.Sprintf("%s/%s", repoPrefix, "config")
			tempDir := PrepareTempFiles()

			ORAS("push", RegistryRef(Host, repo, tag), "--config", foobar.FileConfigName, foobar.FileBarName, "-v").
				MatchStatus([]match.StateKey{
					foobar.FileConfigStateKey,
					foobar.FileBarStateKey,
				}, true, 2).
				WithWorkDir(tempDir).Exec()
			// validate
			fetched := ORAS("manifest", "fetch", RegistryRef(Host, repo, tag)).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Config).Should(Equal(ocispec.Descriptor{
				MediaType: "application/vnd.unknown.config.v1+json",
				Size:      int64(foobar.FileConfigSize),
				Digest:    foobar.FileConfigDigest,
			}))
		})

		It("should push files with customized config file and mediatype", func() {
			repo := fmt.Sprintf("%s/%s", repoPrefix, "config-mediatype")
			configType := "config.type"
			tempDir := PrepareTempFiles()

			ORAS("push", RegistryRef(Host, repo, tag), "--config", fmt.Sprintf("%s:%s", foobar.FileConfigName, configType), foobar.FileBarName, "-v").
				MatchStatus([]match.StateKey{
					{Digest: "46b68ac1696c", Name: configType},
					foobar.FileBarStateKey,
				}, true, 2).
				WithWorkDir(tempDir).Exec()
			// validate
			fetched := ORAS("manifest", "fetch", RegistryRef(Host, repo, tag)).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Config).Should(Equal(ocispec.Descriptor{
				MediaType: configType,
				Size:      int64(foobar.FileConfigSize),
				Digest:    foobar.FileConfigDigest,
			}))
		})

		It("should push files with customized manifest annotation", func() {
			repo := fmt.Sprintf("%s/%s", repoPrefix, "manifest-annotation")
			key := "image-anno-key"
			value := "image-anno-value"
			tempDir := PrepareTempFiles()
			// test
			ORAS("push", RegistryRef(Host, repo, tag), foobar.FileBarName, "-v", "--annotation", fmt.Sprintf("%s=%s", key, value)).
				MatchStatus(statusKeys, true, len(statusKeys)).
				WithWorkDir(tempDir).Exec()
			// validate
			fetched := ORAS("manifest", "fetch", RegistryRef(Host, repo, tag)).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Annotations[key]).To(Equal(value))
		})

		It("should push files with customized file annotation", func() {
			repo := fmt.Sprintf("%s/%s", repoPrefix, "file-annotation")
			tempDir := PrepareTempFiles()

			ORAS("push", RegistryRef(Host, repo, tag), foobar.FileBarName, "-v", "--annotation-file", "foobar/annotation.json", "--config", foobar.FileConfigName).
				MatchStatus(statusKeys, true, 1).
				WithWorkDir(tempDir).Exec()

			// validate
			// see testdata\files\foobar\annotation.json
			fetched := ORAS("manifest", "fetch", RegistryRef(Host, repo, tag)).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Annotations["hi"]).To(Equal("manifest"))
			Expect(manifest.Config.Annotations["hello"]).To(Equal("config"))
			Expect(len(manifest.Layers)).To(Equal(1))
			Expect(manifest.Layers[0].Annotations["foo"]).To(Equal("bar"))
		})
	})
})

var _ = Describe("OCI image layout users:", func() {
	tag := "e2e"
	When("pushing to registy without OCI artifact support", func() {
		statusKeys := []match.StateKey{
			foobar.ImageConfigStateKey("application/vnd.unknown.config.v1+json"),
			foobar.FileBarStateKey,
		}

		It("should push files without customized media types", func() {
			tempDir := PrepareTempFiles()
			ref := LayoutRef(tempDir, tag)
			// test
			ORAS("push", Flags.Layout, ref, foobar.FileBarName, "-v").
				MatchStatus(statusKeys, true, len(statusKeys)).
				WithWorkDir(tempDir).Exec()
			// validate
			fetched := ORAS("manifest", "fetch", Flags.Layout, ref).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Layers).Should(ContainElements(foobar.BlobBarDescriptor("application/vnd.oci.image.layer.v1.tar")))
		})

		It("should push files and tag", func() {
			tempDir := PrepareTempFiles()
			ref := LayoutRef(tempDir, tag)
			extraTag := "2e2"

			ORAS("push", Flags.Layout, fmt.Sprintf("%s,%s", ref, extraTag), foobar.FileBarName, "-v").
				MatchStatus(statusKeys, true, len(statusKeys)).
				WithWorkDir(tempDir).Exec()

			// validate
			fetched := ORAS("manifest", "fetch", Flags.Layout, ref).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Layers).Should(ContainElements(foobar.BlobBarDescriptor("application/vnd.oci.image.layer.v1.tar")))

			fetched = ORAS("manifest", "fetch", Flags.Layout, LayoutRef(tempDir, extraTag)).Exec().Out.Contents()
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Layers).Should(ContainElements(foobar.BlobBarDescriptor("application/vnd.oci.image.layer.v1.tar")))
		})

		It("should push files with customized media types", func() {
			layerType := "layer.type"
			tempDir := PrepareTempFiles()
			ref := LayoutRef(tempDir, tag)
			ORAS("push", Flags.Layout, ref, foobar.FileBarName+":"+layerType, "-v").
				MatchStatus(statusKeys, true, len(statusKeys)).
				WithWorkDir(tempDir).Exec()
			// validate
			fetched := ORAS("manifest", "fetch", Flags.Layout, ref).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Layers).Should(ContainElements(foobar.BlobBarDescriptor(layerType)))
		})

		It("should push files with manifest exported", func() {
			tempDir := PrepareTempFiles()
			layerType := "layer.type"
			exportPath := "packed.json"
			ref := LayoutRef(tempDir, tag)
			ORAS("push", ref, Flags.Layout, foobar.FileBarName+":"+layerType, "-v", "--export-manifest", exportPath).
				MatchStatus(statusKeys, true, len(statusKeys)).
				WithWorkDir(tempDir).Exec()
			// validate
			fetched := ORAS("manifest", "fetch", ref, Flags.Layout).Exec().Out.Contents()
			MatchFile(filepath.Join(tempDir, exportPath), string(fetched), DefaultTimeout)
		})

		It("should push files with customized config file", func() {
			tempDir := PrepareTempFiles()
			ref := LayoutRef(tempDir, tag)
			ORAS("push", Flags.Layout, ref, "--config", foobar.FileConfigName, foobar.FileBarName, "-v").
				MatchStatus([]match.StateKey{
					foobar.FileConfigStateKey,
					foobar.FileBarStateKey,
				}, true, 2).
				WithWorkDir(tempDir).Exec()
			// validate
			fetched := ORAS("manifest", "fetch", Flags.Layout, ref).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Config).Should(Equal(ocispec.Descriptor{
				MediaType: "application/vnd.unknown.config.v1+json",
				Size:      int64(foobar.FileConfigSize),
				Digest:    foobar.FileConfigDigest,
			}))
		})

		It("should push files with customized config file and mediatype", func() {
			configType := "config.type"
			tempDir := PrepareTempFiles()
			ref := LayoutRef(tempDir, tag)
			ORAS("push", Flags.Layout, ref, "--config", fmt.Sprintf("%s:%s", foobar.FileConfigName, configType), foobar.FileBarName, "-v").
				MatchStatus([]match.StateKey{
					{Digest: "46b68ac1696c", Name: configType},
					foobar.FileBarStateKey,
				}, true, 2).
				WithWorkDir(tempDir).Exec()
			// validate
			fetched := ORAS("manifest", "fetch", Flags.Layout, ref).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Config).Should(Equal(ocispec.Descriptor{
				MediaType: configType,
				Size:      int64(foobar.FileConfigSize),
				Digest:    foobar.FileConfigDigest,
			}))
		})

		It("should push files with customized manifest annotation", func() {
			tempDir := PrepareTempFiles()
			ref := LayoutRef(tempDir, tag)
			key := "image-anno-key"
			value := "image-anno-value"
			// test
			ORAS("push", Flags.Layout, ref, foobar.FileBarName, "-v", "--annotation", fmt.Sprintf("%s=%s", key, value)).
				MatchStatus(statusKeys, true, len(statusKeys)).
				WithWorkDir(tempDir).Exec()
			// validate
			fetched := ORAS("manifest", "fetch", ref, Flags.Layout).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Annotations[key]).To(Equal(value))
		})

		It("should push files with customized file annotation", func() {
			tempDir := PrepareTempFiles()
			ref := LayoutRef(tempDir, tag)
			// test
			ORAS("push", ref, Flags.Layout, foobar.FileBarName, "-v", "--annotation-file", "foobar/annotation.json", "--config", foobar.FileConfigName).
				MatchStatus(statusKeys, true, 1).
				WithWorkDir(tempDir).Exec()

			// validate
			// see testdata\files\foobar\annotation.json
			fetched := ORAS("manifest", "fetch", ref, Flags.Layout).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Annotations["hi"]).To(Equal("manifest"))
			Expect(manifest.Config.Annotations["hello"]).To(Equal("config"))
			Expect(len(manifest.Layers)).To(Equal(1))
			Expect(manifest.Layers[0].Annotations["foo"]).To(Equal("bar"))
		})
	})
})
