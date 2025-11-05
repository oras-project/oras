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
	. "oras.land/oras/test/e2e/internal/utils"
)

var _ = Describe("ORAS user:", func() {
	When("generating documentation", func() {
		var tempDir string

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "oras-e2e-docs-*")
			Expect(err).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(tempDir)
		})

		It("should generate markdown documentation", func() {
			ORAS("docs", "--type", "markdown", "--dir", tempDir).Exec()

			// Check that markdown files were created
			entries, err := os.ReadDir(tempDir)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(len(entries)).Should(BeNumerically(">", 0))

			// Verify at least one .md file exists
			foundMd := false
			for _, entry := range entries {
				if filepath.Ext(entry.Name()) == ".md" {
					foundMd = true
					break
				}
			}
			Expect(foundMd).Should(BeTrue())
		})

		It("should generate markdown documentation with short type name", func() {
			ORAS("docs", "--type", "md", "--dir", tempDir).Exec()

			// Check that markdown files were created
			entries, err := os.ReadDir(tempDir)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(len(entries)).Should(BeNumerically(">", 0))
		})

		It("should generate markdown documentation with headers", func() {
			ORAS("docs", "--type", "markdown", "--generate-headers", "--dir", tempDir).Exec()

			// Check that markdown files were created with headers
			entries, err := os.ReadDir(tempDir)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(len(entries)).Should(BeNumerically(">", 0))

			// Read one of the markdown files and verify it has a header
			for _, entry := range entries {
				if filepath.Ext(entry.Name()) == ".md" {
					content, err := os.ReadFile(filepath.Join(tempDir, entry.Name()))
					Expect(err).ShouldNot(HaveOccurred())
					contentStr := string(content)
					Expect(len(contentStr)).Should(BeNumerically(">=", 3))
					Expect(contentStr[:3]).Should(Equal("---"))
					break
				}
			}
		})

		It("should generate man pages", func() {
			ORAS("docs", "--type", "man", "--dir", tempDir).Exec()

			// Check that man files were created
			entries, err := os.ReadDir(tempDir)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(len(entries)).Should(BeNumerically(">", 0))
		})

		It("should generate bash completions", func() {
			ORAS("docs", "--type", "bash", "--dir", tempDir).Exec()

			// Check that completions.bash was created
			completionFile := filepath.Join(tempDir, "completions.bash")
			_, err := os.Stat(completionFile)
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("should fail with invalid documentation type", func() {
			ORAS("docs", "--type", "invalid", "--dir", tempDir).ExpectFailure().Exec()
		})

		It("should use current directory by default", func() {
			// Use a subdirectory as working directory
			workDir := filepath.Join(tempDir, "work")
			err := os.Mkdir(workDir, 0755)
			Expect(err).ShouldNot(HaveOccurred())

			ORAS("docs", "--type", "bash").WithWorkDir(workDir).Exec()

			// Check that completions.bash was created in working directory
			completionFile := filepath.Join(workDir, "completions.bash")
			_, err = os.Stat(completionFile)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})
