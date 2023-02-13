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
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	. "oras.land/oras/test/e2e/internal/utils"
	"oras.land/oras/test/e2e/internal/utils/match"
)

var _ = Describe("ORAS beginners:", func() {
	When("running discover command", func() {
		RunAndShowPreviewInHelp([]string{"discover"})

		It("should show preview and help doc", func() {
			ORAS("discover", "--help").MatchKeyWords("[Preview] Discover", PreviewDesc, ExampleDesc).Exec()
		})

		It("should fail when no subject reference provided", func() {
			ORAS("discover").ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})

		It("should fail when no tag or digest found in provided subject reference", func() {
			ORAS("discover", Reference(Host, Repo, "")).ExpectFailure().MatchErrKeyWords("Error:", "invalid image reference").Exec()
		})
	})
})

var _ = Describe("Common registry users:", func() {
	When("running discover command", Focus, func() {
		It("should discover direct referrers of a subject in remote registry", func() {
			subjectRef := Reference(Host, ArtifactRepo, FoobarImageTag)

			bytes := ORAS("discover", subjectRef, "-o", "json").Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(len(index.Manifests)).To(Equal(2))
			Expect(index.Manifests[0].MediaType).To(Equal(ocispec.MediaTypeImageManifest))
			Expect(index.Manifests[0].Digest.String()).To(Equal(SBOMImageReferrerDigest))
		})
	})
})

var _ = Describe("Fallback registry users:", func() {
	When("running attach command", func() {
		It("should attach a file via a OCI Image", func() {
			testRepo := attachTestRepo("fallback/image")
			tempDir := CopyTestDataToTemp()
			subjectRef := Reference(FallbackHost, testRepo, FoobarImageTag)
			prepare(Reference(FallbackHost, ArtifactRepo, FoobarImageTag), subjectRef)
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
			subjectRef := Reference(FallbackHost, testRepo, FoobarImageTag)
			prepare(Reference(FallbackHost, ArtifactRepo, FoobarImageTag), subjectRef)
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
			subjectRef := Reference(FallbackHost, testRepo, FoobarImageTag)
			prepare(Reference(FallbackHost, ArtifactRepo, FoobarImageTag), subjectRef)
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
