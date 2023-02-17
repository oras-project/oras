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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	. "oras.land/oras/test/e2e/internal/utils"
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
	When("running discover command with json output", func() {
		format := "json"
		It("should discover direct referrers of a subject", func() {
			subjectRef := Reference(Host, ArtifactRepo, foobar.Tag)
			bytes := ORAS("discover", subjectRef, "-o", format).Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(index.Manifests).To(HaveLen(2))
			Expect(index.Manifests).Should(ContainElement(foobar.SBOMImageReferrer))
			Expect(index.Manifests).Should(ContainElement(foobar.SBOMArtifactReferrer))
		})
	})

	When("running discover command with tree output", func() {
		format := "tree"
		It("should discover all referrers of a subject", func() {
			subjectRef := Reference(Host, ArtifactRepo, foobar.Tag)
			ORAS("discover", subjectRef, "-o", format).
				MatchKeyWords(
					Reference(Host, ArtifactRepo, foobar.Digest),
					foobar.SBOMImageReferrer.Digest.String(),
					foobar.SBOMImageReferrer.ArtifactType,
					foobar.SBOMArtifactReferrer.Digest.String(),
					foobar.SBOMArtifactReferrer.ArtifactType,
					foobar.SignatureImageReferrer.Digest.String(),
					foobar.SignatureImageReferrer.ArtifactType,
					foobar.SignatureArtifactReferrer.Digest.String(),
					foobar.SignatureArtifactReferrer.ArtifactType,
				).Exec()
		})
	})

	When("running discover command with table output", func() {
		format := "table"
		It("should all referrers of a subject", func() {
			subjectRef := Reference(Host, ArtifactRepo, foobar.Tag)
			ORAS("discover", subjectRef, "-o", format).
				MatchKeyWords(
					foobar.Digest,
					foobar.SBOMImageReferrer.Digest.String(),
					foobar.SBOMImageReferrer.ArtifactType,
					foobar.SBOMArtifactReferrer.Digest.String(),
					foobar.SBOMArtifactReferrer.ArtifactType,
				).Exec()
		})
	})
})

var _ = Describe("Fallback registry users:", func() {
	When("running discover command", func() {
		It("should discover direct referrers of a subject via json output", func() {
			subjectRef := Reference(FallbackHost, ArtifactRepo, foobar.Tag)
			bytes := ORAS("discover", subjectRef, "-o", "json").Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(index.Manifests).To(HaveLen(1))
			Expect(index.Manifests).Should(ContainElement(foobar.FallbackSBOMImageReferrer))
		})

		It("should discover all referrers of a subject", func() {
			subjectRef := Reference(Host, ArtifactRepo, foobar.Tag)
			ORAS("discover", subjectRef, "-o", "tree").
				MatchKeyWords(
					Reference(Host, ArtifactRepo, foobar.Digest),
					foobar.SBOMImageReferrer.Digest.String(),
					foobar.SBOMImageReferrer.ArtifactType,
					foobar.SBOMArtifactReferrer.Digest.String(),
					foobar.SBOMArtifactReferrer.ArtifactType,
					foobar.SignatureImageReferrer.Digest.String(),
					foobar.SignatureImageReferrer.ArtifactType,
					foobar.SignatureArtifactReferrer.Digest.String(),
					foobar.SignatureArtifactReferrer.ArtifactType,
				).Exec()
		})
	})
})
