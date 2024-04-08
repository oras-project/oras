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
	"oras.land/oras/test/e2e/internal/testdata/artifact"
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

		It("should fail and show detailed error description if no argument provided", func() {
			err := ORAS("push").ExpectFailure().Exec().Err
			gomega.Expect(err).Should(gbytes.Say("Error"))
			gomega.Expect(err).Should(gbytes.Say("\nUsage: oras push"))
			gomega.Expect(err).Should(gbytes.Say("\n"))
			gomega.Expect(err).Should(gbytes.Say(`Run "oras push -h"`))
		})

		It("should fail if the provided reference is not valid", func() {
			err := ORAS("push", "/oras").ExpectFailure().Exec().Err
			gomega.Expect(err).Should(gbytes.Say(`Error: "/oras": invalid reference: invalid registry ""`))
			gomega.Expect(err).Should(gbytes.Say(regexp.QuoteMeta("Please make sure the provided reference is in the form of <registry>/<repo>[:tag|@digest]")))
		})

		It("should fail if the to-be-pushed file is not found", func() {
			tempDir := GinkgoT().TempDir()
			notFoundFilePath := "file/not/found"
			err := ORAS("push", RegistryRef(ZOTHost, pushTestRepo("file-not-found"), ""), notFoundFilePath).
				WithWorkDir(tempDir).
				ExpectFailure().Exec().Err
			gomega.Expect(err).Should(gbytes.Say(filepath.Join(tempDir, notFoundFilePath)))
			gomega.Expect(err).Should(gbytes.Say("no such file or directory"))
		})

		It("should fail to use --config and --artifact-type at the same time for OCI spec v1.0 registry", func() {
			tempDir := PrepareTempFiles()
			repo := pushTestRepo("no-mediatype")
			ref := RegistryRef(ZOTHost, repo, "")

			ORAS("push", ref, "--config", foobar.FileConfigName, "--artifact-type", "test/artifact+json", "--image-spec", "v1.0").ExpectFailure().WithWorkDir(tempDir).Exec()
		})

		It("should fail if image spec is not valid", func() {
			testRepo := attachTestRepo("invalid-image-spec")
			subjectRef := RegistryRef(ZOTHost, testRepo, foobar.Tag)
			invalidFlag := "???"
			ORAS("push", subjectRef, Flags.ImageSpec, invalidFlag).
				ExpectFailure().
				MatchErrKeyWords("Error:", invalidFlag, "Available options: v1.1, v1.0").
				Exec()
		})

		It("should fail if image spec is not valid", func() {
			testRepo := attachTestRepo("invalid-image-spec")
			subjectRef := RegistryRef(ZOTHost, testRepo, foobar.Tag)
			invalidFlag := "???"
			ORAS("push", subjectRef, Flags.ImageSpec, invalidFlag).
				ExpectFailure().
				MatchErrKeyWords("Error:", invalidFlag, "Available options: v1.1, v1.0").
				Exec()
		})

		It("should fail if image spec is not valid", func() {
			testRepo := attachTestRepo("invalid-image-spec")
			subjectRef := RegistryRef(ZOTHost, testRepo, foobar.Tag)
			invalidFlag := "???"
			ORAS("push", subjectRef, Flags.ImageSpec, invalidFlag).
				ExpectFailure().
				MatchErrKeyWords("Error:", invalidFlag, "Available options: v1.1, v1.0").
				Exec()
		})
	})
})

func pushTestRepo(text string) string {
	return fmt.Sprintf("command/push/%d/%s", GinkgoRandomSeed(), text)
}

var _ = Describe("Remote registry users:", func() {
	tag := "e2e"
	When("pushing to OCI spec v1.0 registries", func() {
		statusKeys := []match.StateKey{
			foobar.ImageConfigStateKey("application/vnd.oci.empty.v1+json"),
			foobar.FileBarStateKey,
		}
		It("should push files without customized media types", func() {
			repo := pushTestRepo("no-mediatype")
			tempDir := PrepareTempFiles()
			ref := RegistryRef(ZOTHost, repo, tag)

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
			repo := pushTestRepo("disable-path-validation")
			ref := RegistryRef(ZOTHost, repo, tag)
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
			repo := pushTestRepo("path-validation")
			ref := RegistryRef(ZOTHost, repo, tag)
			absBarName := filepath.Join(PrepareTempFiles(), foobar.FileBarName)
			// test
			ORAS("push", ref, absBarName, "-v").
				MatchErrKeyWords("--disable-path-validation").
				ExpectFailure().
				Exec()
		})

		It("should push files and tag", func() {
			repo := pushTestRepo("multi-tag")
			tempDir := PrepareTempFiles()
			extraTag := "2e2"

			ORAS("push", fmt.Sprintf("%s,%s", RegistryRef(ZOTHost, repo, tag), extraTag), foobar.FileBarName, "-v").
				MatchStatus(statusKeys, true, len(statusKeys)).
				WithWorkDir(tempDir).Exec()

			// validate
			fetched := ORAS("manifest", "fetch", RegistryRef(ZOTHost, repo, tag)).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Layers).Should(ContainElements(foobar.BlobBarDescriptor("application/vnd.oci.image.layer.v1.tar")))

			fetched = ORAS("manifest", "fetch", RegistryRef(ZOTHost, repo, extraTag)).Exec().Out.Contents()
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Layers).Should(ContainElements(foobar.BlobBarDescriptor("application/vnd.oci.image.layer.v1.tar")))
		})

		It("should push files with customized media types", func() {
			repo := pushTestRepo("layer-mediatype")
			layerType := "layer/type"
			tempDir := PrepareTempFiles()
			ORAS("push", RegistryRef(ZOTHost, repo, tag), foobar.FileBarName+":"+layerType, "-v").
				MatchStatus(statusKeys, true, len(statusKeys)).
				WithWorkDir(tempDir).Exec()
			// validate
			fetched := ORAS("manifest", "fetch", RegistryRef(ZOTHost, repo, tag)).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Layers).Should(ContainElements(foobar.BlobBarDescriptor(layerType)))
		})

		It("should push files with manifest exported", func() {
			repo := pushTestRepo("export-manifest")
			layerType := "layer/type"
			tempDir := PrepareTempFiles()
			exportPath := "packed.json"
			ORAS("push", RegistryRef(ZOTHost, repo, tag), foobar.FileBarName+":"+layerType, "-v", "--export-manifest", exportPath).
				MatchStatus(statusKeys, true, len(statusKeys)).
				WithWorkDir(tempDir).Exec()
			// validate
			fetched := ORAS("manifest", "fetch", RegistryRef(ZOTHost, repo, tag)).Exec().Out.Contents()
			MatchFile(filepath.Join(tempDir, exportPath), string(fetched), DefaultTimeout)
		})

		It("should push files with customized config file", func() {
			repo := pushTestRepo("config")
			tempDir := PrepareTempFiles()

			ORAS("push", RegistryRef(ZOTHost, repo, tag), "--config", foobar.FileConfigName, foobar.FileBarName, "-v").
				MatchStatus([]match.StateKey{
					foobar.FileConfigStateKey,
					foobar.FileBarStateKey,
				}, true, 2).
				WithWorkDir(tempDir).Exec()
			// validate
			fetched := ORAS("manifest", "fetch", RegistryRef(ZOTHost, repo, tag)).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Config).Should(Equal(ocispec.Descriptor{
				MediaType: "application/vnd.unknown.config.v1+json",
				Size:      int64(foobar.FileConfigSize),
				Digest:    foobar.FileConfigDigest,
			}))
		})

		It("should push files with customized config file and mediatype", func() {
			repo := pushTestRepo("config/mediatype")
			configType := "config/type"
			tempDir := PrepareTempFiles()

			ORAS("push", RegistryRef(ZOTHost, repo, tag), "--config", fmt.Sprintf("%s:%s", foobar.FileConfigName, configType), foobar.FileBarName, "-v").
				MatchStatus([]match.StateKey{
					{Digest: foobar.FileConfigStateKey.Digest, Name: configType},
					foobar.FileBarStateKey,
				}, true, 2).
				WithWorkDir(tempDir).Exec()
			// validate
			fetched := ORAS("manifest", "fetch", RegistryRef(ZOTHost, repo, tag)).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Config).Should(Equal(ocispec.Descriptor{
				MediaType: configType,
				Size:      int64(foobar.FileConfigSize),
				Digest:    foobar.FileConfigDigest,
			}))
		})

		It("should push files with customized manifest annotation", func() {
			repo := pushTestRepo("manifest-annotation")
			key := "image-anno-key"
			value := "image-anno-value"
			tempDir := PrepareTempFiles()
			// test
			ORAS("push", RegistryRef(ZOTHost, repo, tag), foobar.FileBarName, "-v", "--annotation", fmt.Sprintf("%s=%s", key, value)).
				MatchStatus(statusKeys, true, len(statusKeys)).
				WithWorkDir(tempDir).Exec()
			// validate
			fetched := ORAS("manifest", "fetch", RegistryRef(ZOTHost, repo, tag)).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Annotations[key]).To(Equal(value))
		})

		It("should push files with customized file annotation", func() {
			repo := pushTestRepo("file-annotation")
			tempDir := PrepareTempFiles()

			ORAS("push", RegistryRef(ZOTHost, repo, tag), foobar.FileBarName, "-v", "--annotation-file", "foobar/annotation.json", "--config", foobar.FileConfigName).
				MatchStatus(statusKeys, true, 1).
				WithWorkDir(tempDir).Exec()

			// validate
			// see testdata\files\foobar\annotation.json
			fetched := ORAS("manifest", "fetch", RegistryRef(ZOTHost, repo, tag)).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Annotations["hi"]).To(Equal("manifest"))
			Expect(manifest.Config.Annotations["hello"]).To(Equal("config"))
			Expect(len(manifest.Layers)).To(Equal(1))
			Expect(manifest.Layers[0].Annotations["foo"]).To(Equal("bar"))
		})
	})

	When("pushing to OCI spec v1.1 registries", func() {
		It("should push artifact without layer", func() {
			repo := pushTestRepo("artifact-no-layer")
			tempDir := PrepareTempFiles()
			artifactType := "test/artifact+json"
			annotationKey := "key"
			annotationValue := "value"

			// test
			ORAS("push", RegistryRef(ZOTHost, repo, tag), "-a", fmt.Sprintf("%s=%s", annotationKey, annotationValue), "-v", "--artifact-type", artifactType).
				MatchStatus([]match.StateKey{artifact.DefaultConfigStateKey}, true, 1).
				WithWorkDir(tempDir).Exec()

			// validate
			fetched := ORAS("manifest", "fetch", RegistryRef(ZOTHost, repo, tag)).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.ArtifactType).Should(Equal(artifactType))
			Expect(manifest.Layers).Should(HaveLen(1))
			Expect(manifest.Layers[0]).Should(Equal(artifact.EmptyLayerJSON))
			Expect(manifest.Config).Should(Equal(artifact.EmptyLayerJSON))
			Expect(manifest.Annotations).NotTo(BeNil())
			Expect(manifest.Annotations[annotationKey]).Should(Equal(annotationValue))
		})

		It("should push artifact and format reference", func() {
			repo := pushTestRepo("format-go-template")
			tempDir := PrepareTempFiles()
			annotationKey := "key"
			annotationValue := "value"

			// test
			out := ORAS("push", RegistryRef(ZOTHost, repo, tag), "-a", fmt.Sprintf("%s=%s", annotationKey, annotationValue), "--format", "{{.Ref}}").
				WithWorkDir(tempDir).Exec().Out

			// validate
			ref := string(out.Contents())
			fetched := ORAS("manifest", "fetch", ref).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.Layers).Should(HaveLen(1))
			Expect(manifest.Layers[0]).Should(Equal(artifact.EmptyLayerJSON))
			Expect(manifest.Config).Should(Equal(artifact.EmptyLayerJSON))
			Expect(manifest.Annotations).NotTo(BeNil())
			Expect(manifest.Annotations[annotationKey]).Should(Equal(annotationValue))
		})

		It("should push artifact and format json", func() {
			repo := pushTestRepo("format-json")
			tempDir := PrepareTempFiles()
			artifactType := "test/artifact+json"
			annotationKey := "key"
			annotationValue := "value"

			// test
			out := ORAS("push", RegistryRef(ZOTHost, repo, tag), "-a", fmt.Sprintf("%s=%s", annotationKey, annotationValue), "--format", "json", "--artifact-type", artifactType).
				WithWorkDir(tempDir).Exec().Out

			// validate
			Expect(out).To(gbytes.Say(RegistryRef(ZOTHost, repo, "")))
			Expect(out).To(gbytes.Say(regexp.QuoteMeta(fmt.Sprintf(`"ArtifactType": "%s"`, artifactType))))
		})

		It("should push files", func() {
			repo := pushTestRepo("artifact-with-blob")
			tempDir := PrepareTempFiles()

			ORAS("push", RegistryRef(ZOTHost, repo, tag), foobar.FileBarName, "-v").
				MatchStatus([]match.StateKey{foobar.FileBarStateKey, artifact.DefaultConfigStateKey}, true, 2).
				WithWorkDir(tempDir).Exec()

			// validate
			fetched := ORAS("manifest", "fetch", RegistryRef(ZOTHost, repo, tag)).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.ArtifactType).Should(Equal("application/vnd.unknown.artifact.v1"))
			Expect(manifest.Layers).Should(ContainElements(foobar.BlobBarDescriptor("application/vnd.oci.image.layer.v1.tar")))
			Expect(manifest.Config).Should(Equal(artifact.EmptyLayerJSON))
		})

		It("should push v1.1-rc.4 artifact", func() {
			repo := pushTestRepo("v1.1-artifact")
			tempDir := PrepareTempFiles()

			ORAS("push", RegistryRef(ZOTHost, repo, tag), foobar.FileBarName, "-v", "--image-spec", "v1.1").
				MatchStatus([]match.StateKey{foobar.FileBarStateKey, artifact.DefaultConfigStateKey}, true, 2).
				WithWorkDir(tempDir).Exec()

			// validate
			fetched := ORAS("manifest", "fetch", RegistryRef(ZOTHost, repo, tag)).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.ArtifactType).Should(Equal("application/vnd.unknown.artifact.v1"))
			Expect(manifest.Layers).Should(ContainElements(foobar.BlobBarDescriptor("application/vnd.oci.image.layer.v1.tar")))
			Expect(manifest.Config).Should(Equal(artifact.EmptyLayerJSON))
		})

		It("should push artifact with config", func() {
			repo := pushTestRepo("artifact-with-config")
			tempDir := PrepareTempFiles()
			configType := "test/config+json"

			ORAS("push", RegistryRef(ZOTHost, repo, tag), foobar.FileBarName, "--config", fmt.Sprintf("%s:%s", foobar.FileConfigName, configType), "-v").
				MatchStatus([]match.StateKey{
					foobar.FileBarStateKey,
					{Digest: foobar.FileConfigStateKey.Digest, Name: configType},
					artifact.DefaultConfigStateKey}, true, 2).
				WithWorkDir(tempDir).Exec()

			// validate
			fetched := ORAS("manifest", "fetch", RegistryRef(ZOTHost, repo, tag)).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.ArtifactType).Should(Equal(""))
			Expect(manifest.Layers).Should(ContainElements(foobar.BlobBarDescriptor("application/vnd.oci.image.layer.v1.tar")))
			Expect(manifest.Config.MediaType).Should(Equal(configType))
			Expect(manifest.Config.Digest).Should(Equal(foobar.FileConfigDigest))
		})

		It("should push artifact with artifact type and config data", func() {
			repo := pushTestRepo("artifact-type-and-config")
			tempDir := PrepareTempFiles()
			artifactType := "test/artifact+json"
			configType := "test/config+json"

			ORAS("push", RegistryRef(ZOTHost, repo, tag), foobar.FileBarName, "--artifact-type", artifactType, "--config", fmt.Sprintf("%s:%s", foobar.FileConfigName, configType), "-v").
				MatchStatus([]match.StateKey{
					foobar.FileBarStateKey,
					{Digest: foobar.FileConfigStateKey.Digest, Name: configType},
					artifact.DefaultConfigStateKey}, true, 2).
				WithWorkDir(tempDir).Exec()

			// validate
			fetched := ORAS("manifest", "fetch", RegistryRef(ZOTHost, repo, tag)).Exec().Out.Contents()
			var manifest ocispec.Manifest
			Expect(json.Unmarshal(fetched, &manifest)).ShouldNot(HaveOccurred())
			Expect(manifest.ArtifactType).Should(Equal(artifactType))
			Expect(manifest.Layers).Should(ContainElements(foobar.BlobBarDescriptor("application/vnd.oci.image.layer.v1.tar")))
			Expect(manifest.Config.MediaType).Should(Equal(configType))
			Expect(manifest.Config.Digest).Should(Equal(foobar.FileConfigDigest))
		})
	})
})

var _ = Describe("OCI image layout users:", func() {
	tag := "e2e"
	When("pushing to registry without OCI artifact support", func() {
		statusKeys := []match.StateKey{
			foobar.ImageConfigStateKey("application/vnd.oci.empty.v1+json"),
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
			configType := "config/type"
			tempDir := PrepareTempFiles()
			ref := LayoutRef(tempDir, tag)
			ORAS("push", Flags.Layout, ref, "--config", fmt.Sprintf("%s:%s", foobar.FileConfigName, configType), foobar.FileBarName, "-v").
				MatchStatus([]match.StateKey{
					{Digest: foobar.FileConfigStateKey.Digest, Name: configType},
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
