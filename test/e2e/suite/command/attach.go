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
			gomega.Expect(out).Should(gbytes.Say("--image-spec string\\s+%s", regexp.QuoteMeta(feature.Experimental.Mark)))
		})

		It("should fail when no subject reference provided", func() {
			ORAS("attach", "--artifact-type", "oras.test").ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})

		It("should fail if no file reference or manifest annotation provided", func() {
			ORAS("attach", "--artifact-type", "oras.test", RegistryRef(Host, ImageRepo, foobar.Tag)).
				ExpectFailure().MatchErrKeyWords("Error: no blob or manifest annotation are provided").Exec()
		})
	})
})

var _ = Describe("Common registry users:", func() {
	When("running attach command", func() {
		It("should attach a file to a subject", func() {
			testRepo := attachTestRepo("simple")
			tempDir := PrepareTempFiles()
			subjectRef := RegistryRef(Host, testRepo, foobar.Tag)
			prepare(RegistryRef(Host, ImageRepo, foobar.Tag), subjectRef)
			ORAS("attach", "--artifact-type", "test.attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia)).
				WithWorkDir(tempDir).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()
		})

		It("should attach a file to a subject and export the built manifest", func() {
			// prepare
			testRepo := attachTestRepo("export-manifest")
			tempDir := PrepareTempFiles()
			exportName := "manifest.json"
			subjectRef := RegistryRef(Host, testRepo, foobar.Tag)
			prepare(RegistryRef(Host, ImageRepo, foobar.Tag), subjectRef)
			// test
			ORAS("attach", "--artifact-type", "test.attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--export-manifest", exportName).
				WithWorkDir(tempDir).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()
			// validate
			var index ocispec.Index
			bytes := ORAS("discover", subjectRef, "-o", "json").Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			fetched := ORAS("manifest", "fetch", RegistryRef(Host, testRepo, index.Manifests[0].Digest.String())).Exec().Out.Contents()
			MatchFile(filepath.Join(tempDir, exportName), string(fetched), DefaultTimeout)
		})
		It("should attach a file via a OCI Image", func() {
			testRepo := attachTestRepo("image")
			tempDir := PrepareTempFiles()
			subjectRef := RegistryRef(Host, testRepo, foobar.Tag)
			prepare(RegistryRef(Host, ImageRepo, foobar.Tag), subjectRef)
			// test
			ORAS("attach", "--artifact-type", "test.attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--image-spec", "v1.1-image").
				WithWorkDir(tempDir).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()

			// validate
			var index ocispec.Index
			bytes := ORAS("discover", subjectRef, "-o", "json").Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].MediaType).To(Equal(ocispec.MediaTypeImageManifest))
		})
		It("should attach a file via a OCI Artifact", func() {
			testRepo := attachTestRepo("artifact")
			tempDir := PrepareTempFiles()
			subjectRef := RegistryRef(Host, testRepo, foobar.Tag)
			prepare(RegistryRef(Host, ImageRepo, foobar.Tag), subjectRef)
			// test
			ORAS("attach", "--artifact-type", "test.attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--image-spec", "v1.1-artifact").
				WithWorkDir(tempDir).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()

			// validate
			var index ocispec.Index
			bytes := ORAS("discover", subjectRef, "-o", "json").Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].MediaType).To(Equal(ocispec.MediaTypeArtifactManifest))
		})
	})
})

var _ = Describe("Fallback registry users:", func() {
	When("running attach command", func() {
		It("should attach a file via a OCI Image", func() {
			testRepo := attachTestRepo("fallback/image")
			tempDir := PrepareTempFiles()
			subjectRef := RegistryRef(FallbackHost, testRepo, foobar.Tag)
			prepare(RegistryRef(FallbackHost, ArtifactRepo, foobar.Tag), subjectRef)
			// test
			ORAS("attach", "--artifact-type", "test.attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--image-spec", "v1.1-image").
				WithWorkDir(tempDir).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()

			// validate
			var index ocispec.Index
			bytes := ORAS("discover", subjectRef, "-o", "json").Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].MediaType).To(Equal(ocispec.MediaTypeImageManifest))
		})

		It("should attach a file via a OCI Image by default", func() {
			testRepo := attachTestRepo("fallback/default")
			tempDir := PrepareTempFiles()
			subjectRef := RegistryRef(FallbackHost, testRepo, foobar.Tag)
			prepare(RegistryRef(FallbackHost, ArtifactRepo, foobar.Tag), subjectRef)
			// test
			ORAS("attach", "--artifact-type", "test.attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--image-spec", "v1.1-image").
				WithWorkDir(tempDir).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()

			// validate
			var index ocispec.Index
			bytes := ORAS("discover", subjectRef, "-o", "json").Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].MediaType).To(Equal(ocispec.MediaTypeImageManifest))
		})

		It("should attach a file via a OCI Image and generate referrer via tag schema", func() {
			testRepo := attachTestRepo("fallback/tag_schema")
			tempDir := PrepareTempFiles()
			subjectRef := RegistryRef(FallbackHost, testRepo, foobar.Tag)
			prepare(RegistryRef(FallbackHost, ArtifactRepo, foobar.Tag), subjectRef)
			// test
			ORAS("attach", "--artifact-type", "test.attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--image-spec", "v1.1-image", "--distribution-spec", "v1.1-referrers-tag").
				WithWorkDir(tempDir).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()

			// validate
			var index ocispec.Index
			bytes := ORAS("discover", subjectRef, "--distribution-spec", "v1.1-referrers-tag", "-o", "json").Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].MediaType).To(Equal(ocispec.MediaTypeImageManifest))
		})
	})
})

var _ = Describe("OCI image layout users:", func() {
	When("running attach command", func() {
		prepare := func(root string) {
			ORAS("cp", RegistryRef(Host, ImageRepo, foobar.Tag), Flags.ToLayout, LayoutRef(root, foobar.Tag)).Exec()
		}
		It("should attach a file to a subject", func() {
			root := PrepareTempFiles()
			subjectRef := LayoutRef(root, foobar.Tag)
			prepare(root)
			ORAS("attach", "--artifact-type", "test.attach", Flags.Layout, subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia)).
				WithWorkDir(root).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()
		})

		It("should attach a file to a subject and export the built manifest", func() {
			// prepare
			root := PrepareTempFiles()
			exportName := "manifest.json"
			subjectRef := LayoutRef(root, foobar.Tag)
			prepare(root)
			// test
			ORAS("attach", "--artifact-type", "test.attach", Flags.Layout, subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--export-manifest", exportName).
				WithWorkDir(root).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()
			// validate
			var index ocispec.Index
			bytes := ORAS("discover", Flags.Layout, subjectRef, "-o", "json").Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].MediaType).To(Equal(ocispec.MediaTypeImageManifest))
			fetched := ORAS("manifest", "fetch", Flags.Layout, LayoutRef(root, index.Manifests[0].Digest.String())).Exec().Out.Contents()
			MatchFile(filepath.Join(root, exportName), string(fetched), DefaultTimeout)
		})
		It("should attach a file via a OCI Image", func() {
			root := PrepareTempFiles()
			subjectRef := LayoutRef(root, foobar.Tag)
			prepare(root)
			// test
			ORAS("attach", "--artifact-type", "test.attach", Flags.Layout, subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--image-spec", "v1.1-image").
				WithWorkDir(root).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()

			// validate
			var index ocispec.Index
			bytes := ORAS("discover", subjectRef, Flags.Layout, "-o", "json").Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].MediaType).To(Equal(ocispec.MediaTypeImageManifest))
		})
		It("should attach a file via a OCI Artifact", func() {
			root := PrepareTempFiles()
			subjectRef := LayoutRef(root, foobar.Tag)
			prepare(root)
			// test
			ORAS("attach", "--artifact-type", "test.attach", subjectRef, Flags.Layout, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--image-spec", "v1.1-artifact").
				WithWorkDir(root).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()

			// validate
			var index ocispec.Index
			bytes := ORAS("discover", subjectRef, Flags.Layout, "-o", "json").Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			Expect(index.Manifests[0].MediaType).To(Equal(ocispec.MediaTypeArtifactManifest))
		})
	})
})
