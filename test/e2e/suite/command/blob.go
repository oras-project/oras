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
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	. "oras.land/oras/test/e2e/internal/utils"
)

var _ = Describe("ORAS beginners:", func() {
	When("running blob command", func() {
		runAndShowPreviewInHelp([]string{"blob"})
		runAndShowPreviewInHelp([]string{"blob", "fetch"}, preview_desc, example_desc)

		It("should call sub-commands with aliases", func() {
			ORAS("blob", "get", "--help").
				MatchKeyWords("[Preview] Fetch", preview_desc, example_desc).
				Exec()
		})
		It("should have flag for prettifying JSON output", func() {
			ORAS("blob", "get", "--help").
				MatchKeyWords("--pretty", "prettify JSON").
				Exec()
		})
		It("should fetch manifest with no artifact reference provided", func() {
			ORAS("blob", "fetch").
				WithFailureCheck().
				MatchErrKeyWords("Error:").
				Exec()
		})
	})
})

var _ = Describe("Common registry users:", func() {
	var blobDigest = "sha256:2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae"
	var blobContent = "foo"
	var blobDescriptor = `{"mediaType":"application/octet-stream","digest":"sha256:2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae","size":3}`

	When("running `blob fetch`", func() {
		It("should fetch blob descriptor ", func() {
			ORAS("blob", "fetch", Reference(Host, repo, blobDigest), "--descriptor").
				MatchContent(blobDescriptor).Exec()
		})
		It("should fetch blob content and output to stdout", func() {
			ORAS("blob", "fetch", Reference(Host, repo, blobDigest), "--output", "-").
				MatchContent(blobContent).Exec()
		})
		It("should fetch blob content and output to a file", func() {
			tempDir := GinkgoT().TempDir()
			contentPath := filepath.Join(tempDir, "fetched")
			ORAS("blob", "fetch", Reference(Host, repo, blobDigest), "--output", contentPath).
				WithWorkDir(tempDir).Exec()
			Expect(contentPath).Should(BeAnExistingFile())
			f, err := os.Open(contentPath)
			Expect(err).ShouldNot(HaveOccurred())
			defer f.Close()
			Eventually(gbytes.BufferReader(f)).Should(gbytes.Say(blobContent))
		})
		It("should fetch blob descriptor and output content to a file", func() {
			tempDir := GinkgoT().TempDir()
			contentPath := filepath.Join(tempDir, "fetched")
			ORAS("blob", "fetch", Reference(Host, repo, blobDigest), "--output", contentPath, "--descriptor").
				MatchContent(blobDescriptor).
				WithWorkDir(tempDir).Exec()
			Expect(contentPath).Should(BeAnExistingFile())
			f, err := os.Open(contentPath)
			Expect(err).ShouldNot(HaveOccurred())
			defer f.Close()
			Eventually(gbytes.BufferReader(f)).Should(gbytes.Say(blobContent))
		})
	})

	When("running `blob fetch` with wrong input", func() {
		It("should fail if neither output path nor descriptor flag are not provided", func() {
			ORAS("blob", "fetch", Reference(Host, repo, blobDigest)).
				WithFailureCheck().Exec()
		})

		It("should fail if no digest provided", func() {
			ORAS("blob", "fetch", Reference(Host, repo, "")).
				WithFailureCheck().Exec()
		})

		It("should fail if provided digest doesn't existed", func() {
			ORAS("blob", "fetch", Reference(Host, repo, "sha256:2aaa2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a")).
				WithFailureCheck().Exec()
		})

		It("should fail if output path points to stdout and descriptor flag is provided", func() {
			ORAS("blob", "fetch", Reference(Host, repo, blobDigest), "--descriptor", "--output", "-").
				WithFailureCheck().Exec()
		})

		It("should fail if no reference is provided", func() {
			ORAS("blob", "fetch").
				WithFailureCheck().Exec()
		})
	})
})
