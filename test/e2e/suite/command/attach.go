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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/test/e2e/internal/testdata/feature"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	. "oras.land/oras/test/e2e/internal/utils"
	"oras.land/oras/test/e2e/internal/utils/match"
)

func attachTestRepo(text string) string {
	return fmt.Sprintf("command/attach/%d/%s", GinkgoRandomSeed(), text)
}

var _ = Describe("ORAS beginners:", func() {
	When("running attach command", func() {
		RunAndShowPreviewInHelp([]string{"attach"})

		It("should show preview and help doc", func() {
			out := ORAS("attach", "--help").MatchKeyWords(feature.Preview.Mark+" Attach", feature.Preview.Description, ExampleDesc).Exec()
			gomega.Expect(out).Should(gbytes.Say("--distribution-spec string\\s+%s", regexp.QuoteMeta(feature.Preview.Mark)))
		})

		It("should fail when no subject reference provided", func() {
			ORAS("attach", "--artifact-type", "oras/test").ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})

		It("should fail if no file reference or manifest annotation provided for registry", func() {
			ORAS("attach", "--artifact-type", "oras/test", RegistryRef(ZOTHost, ImageRepo, foobar.Tag)).
				ExpectFailure().MatchErrKeyWords("Error: neither file nor annotation", "Usage:").Exec()
		})

		It("should fail if no file reference or manifest annotation provided for OCI layout", func() {
			root := GinkgoT().TempDir()
			ORAS("attach", "--artifact-type", "oras/test", LayoutRef(root, foobar.Tag), Flags.Layout).
				ExpectFailure().MatchErrKeyWords("Error: neither file nor annotation", "Usage:").Exec()
		})

		It("should fail if distribution spec is unknown", func() {
			ORAS("attach", "--artifact-type", "oras/test", RegistryRef(ZOTHost, ImageRepo, foobar.Tag), "--distribution-spec", "???").
				ExpectFailure().MatchErrKeyWords("unknown distribution specification flag").Exec()
		})

		It("should fail and show detailed error description if no argument provided", func() {
			err := ORAS("attach").ExpectFailure().Exec().Err
			gomega.Expect(err).Should(gbytes.Say("Error"))
			gomega.Expect(err).Should(gbytes.Say("\nUsage: oras attach"))
			gomega.Expect(err).Should(gbytes.Say("\n"))
			gomega.Expect(err).Should(gbytes.Say(`Run "oras attach -h"`))
		})

		It("should fail if distribution spec is not valid", func() {
			testRepo := attachTestRepo("invalid-image-spec")
			CopyZOTRepo(ImageRepo, testRepo)
			subjectRef := RegistryRef(ZOTHost, testRepo, foobar.Tag)
			invalidFlag := "???"
			ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), Flags.DistributionSpec, invalidFlag).
				ExpectFailure().
				WithWorkDir(PrepareTempFiles()).
				MatchErrKeyWords("Error:", invalidFlag, "Available options: v1.1-referrers-tag, v1.1-referrers-api").
				Exec()
		})
	})
})

var _ = Describe("1.1 registry users:", func() {
	When("running attach command", func() {
		It("should attach a file to a subject", func() {
			testRepo := attachTestRepo("simple")
			CopyZOTRepo(ImageRepo, testRepo)
			subjectRef := RegistryRef(ZOTHost, testRepo, foobar.Tag)
			ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia)).
				WithWorkDir(PrepareTempFiles()).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()
		})

		It("should attach a file to a subject and export the built manifest", func() {
			// prepare
			testRepo := attachTestRepo("export-manifest")
			tempDir := PrepareTempFiles()
			exportName := "manifest.json"
			subjectRef := RegistryRef(ZOTHost, testRepo, foobar.Tag)
			CopyZOTRepo(ImageRepo, testRepo)
			// test
			ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--export-manifest", exportName).
				WithWorkDir(tempDir).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()
			// validate
			var index ocispec.Index
			bytes := ORAS("discover", subjectRef, "-o", "json").Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			fetched := ORAS("manifest", "fetch", RegistryRef(ZOTHost, testRepo, index.Manifests[0].Digest.String())).Exec().Out.Contents()
			MatchFile(filepath.Join(tempDir, exportName), string(fetched), DefaultTimeout)
		})

		It("should attach a file to a subject and format the digest reference", func() {
			// prepare
			testRepo := attachTestRepo("format-ref")
			tempDir := PrepareTempFiles()
			exportName := "manifest.json"
			subjectRef := RegistryRef(ZOTHost, testRepo, foobar.Tag)
			CopyZOTRepo(ImageRepo, testRepo)
			// test
			delimitter := "---"
			output := ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--export-manifest", exportName, "--format", fmt.Sprintf("{{.Ref}}%s{{.ArtifactType}}", delimitter)).
				WithWorkDir(tempDir).Exec().Out.Contents()
			ref, artifactType, _ := strings.Cut(string(output), delimitter)
			// validate
			Expect(artifactType).To(Equal("test/attach"))
			fetched := ORAS("manifest", "fetch", ref).Exec().Out.Contents()
			MatchFile(filepath.Join(tempDir, exportName), string(fetched), DefaultTimeout)
		})

		It("should attach a file to a subject and format json", func() {
			// prepare
			testRepo := attachTestRepo("format-json")
			tempDir := PrepareTempFiles()
			exportName := "manifest.json"
			subjectRef := RegistryRef(ZOTHost, testRepo, foobar.Tag)
			CopyZOTRepo(ImageRepo, testRepo)
			// test
			out := ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--export-manifest", exportName, "--format", "json").
				WithWorkDir(tempDir).Exec().Out
			// validate
			Expect(out).To(gbytes.Say(RegistryRef(ZOTHost, testRepo, "")))
		})

		It("should attach a file via a OCI Image", func() {
			testRepo := attachTestRepo("image")
			tempDir := PrepareTempFiles()
			subjectRef := RegistryRef(ZOTHost, testRepo, foobar.Tag)
			CopyZOTRepo(ImageRepo, testRepo)
			// test
			ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia)).
				WithWorkDir(tempDir).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()
			// validate
			var index ocispec.Index
			bytes := ORAS("discover", subjectRef, "-o", "json").Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].MediaType).To(Equal("application/vnd.oci.image.manifest.v1+json"))
		})

		It("should attach file with path validation disabled", func() {
			testRepo := attachTestRepo("simple")
			absAttachFileName := filepath.Join(PrepareTempFiles(), foobar.AttachFileName)

			subjectRef := RegistryRef(ZOTHost, testRepo, foobar.Tag)
			CopyZOTRepo(ImageRepo, testRepo)
			statusKey := foobar.AttachFileStateKey
			statusKey.Name = absAttachFileName
			ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", absAttachFileName, foobar.AttachFileMedia), "--disable-path-validation").
				MatchStatus([]match.StateKey{statusKey}, false, 1).
				Exec()
		})

		It("should fail path validation when attaching file with absolute path", func() {
			testRepo := attachTestRepo("simple")
			absAttachFileName := filepath.Join(PrepareTempFiles(), foobar.AttachFileName)

			subjectRef := RegistryRef(ZOTHost, testRepo, foobar.Tag)
			CopyZOTRepo(ImageRepo, testRepo)
			statusKey := foobar.AttachFileStateKey
			statusKey.Name = absAttachFileName
			ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", absAttachFileName, foobar.AttachFileMedia)).
				ExpectFailure().
				Exec()
		})
	})
})

var _ = Describe("1.0 registry users:", func() {
	When("running attach command", func() {
		It("should attach a file via a OCI Image", func() {
			testRepo := attachTestRepo("fallback/image")
			tempDir := PrepareTempFiles()
			subjectRef := RegistryRef(FallbackHost, testRepo, foobar.Tag)
			prepare(RegistryRef(FallbackHost, ArtifactRepo, foobar.Tag), subjectRef)
			// test
			ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia)).
				WithWorkDir(tempDir).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()

			// validate
			var index ocispec.Index
			bytes := ORAS("discover", subjectRef, "-o", "json").Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].MediaType).To(Equal("application/vnd.oci.image.manifest.v1+json"))
		})

		It("should attach a file via a OCI Image by default", func() {
			testRepo := attachTestRepo("fallback/default")
			tempDir := PrepareTempFiles()
			subjectRef := RegistryRef(FallbackHost, testRepo, foobar.Tag)
			prepare(RegistryRef(FallbackHost, ArtifactRepo, foobar.Tag), subjectRef)
			// test
			ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia)).
				WithWorkDir(tempDir).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()

			// validate
			var index ocispec.Index
			bytes := ORAS("discover", subjectRef, "-o", "json").Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].MediaType).To(Equal("application/vnd.oci.image.manifest.v1+json"))
		})

		It("should attach a file via a OCI Image and generate referrer via tag schema", func() {
			testRepo := attachTestRepo("fallback/tag_schema")
			tempDir := PrepareTempFiles()
			subjectRef := RegistryRef(FallbackHost, testRepo, foobar.Tag)
			prepare(RegistryRef(FallbackHost, ArtifactRepo, foobar.Tag), subjectRef)
			// test
			ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--distribution-spec", "v1.1-referrers-tag").
				WithWorkDir(tempDir).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()

			// validate
			var index ocispec.Index
			bytes := ORAS("discover", subjectRef, "--distribution-spec", "v1.1-referrers-tag", "-o", "json").Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].MediaType).To(Equal("application/vnd.oci.image.manifest.v1+json"))
		})
	})
})

var _ = Describe("OCI image layout users:", func() {
	When("running attach command", func() {
		It("should attach a file to a subject", func() {
			root := PrepareTempOCI(ImageRepo)
			subjectRef := LayoutRef(root, foobar.Tag)
			ORAS("attach", "--artifact-type", "test/attach", Flags.Layout, subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia)).
				WithWorkDir(root).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()
		})

		It("should attach a file to a subject and export the built manifest", func() {
			// prepare
			exportName := "manifest.json"
			root := PrepareTempOCI(ImageRepo)
			subjectRef := LayoutRef(root, foobar.Tag)
			// test
			ORAS("attach", "--artifact-type", "test/attach", Flags.Layout, subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--export-manifest", exportName).
				WithWorkDir(root).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()
			// validate
			var index ocispec.Index
			bytes := ORAS("discover", Flags.Layout, subjectRef, "-o", "json").Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].MediaType).To(Equal("application/vnd.oci.image.manifest.v1+json"))
			fetched := ORAS("manifest", "fetch", Flags.Layout, LayoutRef(root, index.Manifests[0].Digest.String())).Exec().Out.Contents()
			MatchFile(filepath.Join(root, exportName), string(fetched), DefaultTimeout)
		})
		It("should attach a file via a OCI Image", func() {
			root := PrepareTempOCI(ImageRepo)
			subjectRef := LayoutRef(root, foobar.Tag)
			// test
			ORAS("attach", "--artifact-type", "test/attach", Flags.Layout, subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia)).
				WithWorkDir(root).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()

			// validate
			var index ocispec.Index
			bytes := ORAS("discover", subjectRef, Flags.Layout, "-o", "json").Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].MediaType).To(Equal("application/vnd.oci.image.manifest.v1+json"))
		})
	})
})
