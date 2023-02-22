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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
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
			ORAS("attach", "--help").MatchKeyWords("[Preview] Attach", PreviewDesc, ExampleDesc).Exec()
		})

		It("should fail when no subject reference provided", func() {
			ORAS("attach", "--artifact-type", "oras.test").ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})

		It("should fail if no file reference or manifest annotation provided", func() {
			ORAS("attach", "--artifact-type", "oras.test", Reference(Host, ImageRepo, foobar.Tag)).
				ExpectFailure().MatchErrKeyWords("Error: no blob or manifest annotation are provided").Exec()
		})
	})
})

var _ = Describe("Common registry users:", func() {
	When("running attach command", Focus, func() {
		It("should attach a file to a subject", func() {
			testRepo := attachTestRepo("simple")
			tempDir := CopyTestDataToTemp()
			subjectRef := Reference(Host, testRepo, foobar.Tag)
			prepare(Reference(Host, ImageRepo, foobar.Tag), subjectRef)
			ORAS("attach", "--artifact-type", "test.attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia)).
				WithWorkDir(tempDir).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()
		})

		if strings.HasPrefix(Host, "localhost:") {
			It("should attach a file to a subject via resolve flag", func() {
				testRepo := attachTestRepo("simple-resolve")
				tempDir := CopyTestDataToTemp()
				subjectRefWithFlags := fmt.Sprint(Reference(MockedHost, testRepo, FoobarImageTag), ResolveFlags(Host, MockedHost))
				prepare(Reference(Host, ImageRepo, FoobarImageTag), subjectRefWithFlags)
				ORAS("attach", "--artifact-type", "test.attach", subjectRefWithFlags, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia)).
					WithWorkDir(tempDir).
					MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()
				// validate
				var index ocispec.Index
				bytes := ORAS("discover", subjectRefWithFlags, "-o", "json").Exec().Out.Contents()
				Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
				Expect(len(index.Manifests)).To(Equal(1))
			})
		}

		It("should attach a file to a subject and export the built manifest", func() {
			// prepare
			testRepo := attachTestRepo("export-manifest")
			tempDir := CopyTestDataToTemp()
			exportName := "manifest.json"
			subjectRef := Reference(Host, testRepo, foobar.Tag)
			prepare(Reference(Host, ImageRepo, foobar.Tag), subjectRef)
			// test
			ORAS("attach", "--artifact-type", "test.attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--export-manifest", exportName).
				WithWorkDir(tempDir).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()
			// validate
			var index ocispec.Index
			bytes := ORAS("discover", subjectRef, "-o", "json").Exec().Out.Contents()
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(1))
			fetched := ORAS("manifest", "fetch", Reference(Host, testRepo, index.Manifests[0].Digest.String())).Exec().Out.Contents()
			MatchFile(filepath.Join(tempDir, exportName), string(fetched), DefaultTimeout)
		})
		It("should attach a file via a OCI Image", func() {
			testRepo := attachTestRepo("image")
			tempDir := CopyTestDataToTemp()
			subjectRef := Reference(Host, testRepo, foobar.Tag)
			prepare(Reference(Host, ImageRepo, foobar.Tag), subjectRef)
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
			tempDir := CopyTestDataToTemp()
			subjectRef := Reference(Host, testRepo, foobar.Tag)
			prepare(Reference(Host, ImageRepo, foobar.Tag), subjectRef)
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
			tempDir := CopyTestDataToTemp()
			subjectRef := Reference(FallbackHost, testRepo, foobar.Tag)
			prepare(Reference(FallbackHost, ArtifactRepo, foobar.Tag), subjectRef)
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
			tempDir := CopyTestDataToTemp()
			subjectRef := Reference(FallbackHost, testRepo, foobar.Tag)
			prepare(Reference(FallbackHost, ArtifactRepo, foobar.Tag), subjectRef)
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
			tempDir := CopyTestDataToTemp()
			subjectRef := Reference(FallbackHost, testRepo, foobar.Tag)
			prepare(Reference(FallbackHost, ArtifactRepo, foobar.Tag), subjectRef)
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
