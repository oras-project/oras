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
		It("should fetch manifest with no artifact reference provided", func() {
			ORAS("blob", "fetch").
				WithFailureCheck().
				MatchErrKeyWords("Error:").
				Exec()
		})
	})
})

var _ = Describe("Common registry users:", Focus, func() {
	var repo = "command/images"
	var blobDigest = "sha256:fe9dbc99451d0517d65e048c309f0b5afb2cc513b7a3d456b6cc29fe641386c5"
	var blobContent = `{
    "architecture": "amd64",
    "os": "linux"
}`
	var blobDescriptor = `{"mediaType":"application/octet-stream","digest":"sha256:fe9dbc99451d0517d65e048c309f0b5afb2cc513b7a3d456b6cc29fe641386c5","size":53}`

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
})
