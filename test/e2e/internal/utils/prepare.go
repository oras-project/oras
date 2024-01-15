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

package utils

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

// CopyZOTRepo copies oci layout data between repostories.
func CopyZOTRepo(fromRepo string, toRepo string) {
	zotRoot := filepath.Join(TestDataRoot, "zot")
	fromRepo = filepath.Join(zotRoot, fromRepo)
	toRepo = filepath.Join(zotRoot, toRepo)
	Expect(CopyFiles(fromRepo, toRepo)).ShouldNot(HaveOccurred())
}

// PrepareTempOCI prepares an OCI layout root via copying from an ZOT repo and
// return the path.
func PrepareTempOCI(fromZotRepo string) string {
	root := PrepareTempFiles()
	Expect(CopyFiles(filepath.Join(TestDataRoot, "zot", fromZotRepo), root)).ShouldNot(HaveOccurred())
	return root
}

// PrepareTempFiles copies test data into a temp folder and return it.
func PrepareTempFiles() string {
	tempDir := GinkgoT().TempDir()
	Expect(CopyTestFiles(tempDir)).ShouldNot(HaveOccurred())
	return tempDir
}

// CopyTestFiles copies test data into dstRoot.
func CopyTestFiles(dstRoot string) error {
	return CopyFiles(filepath.Join(TestDataRoot, "files"), dstRoot)
}

// CopyFiles copies files from folder src to folder dest.
func CopyFiles(src string, dest string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// ignore folder
			return nil
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dest, relPath)
		// make sure all parents are created
		if err := os.MkdirAll(filepath.Dir(dstPath), 0700); err != nil {
			return err
		}

		// copy with original folder structure
		return copyFile(path, dstPath)
	})
}

// MatchFile reads content from filepath, matches it with want with timeout.
func MatchFile(filepath string, want string, timeout time.Duration) {
	Expect(filepath).To(BeAnExistingFile())
	f, err := os.Open(filepath)
	Expect(err).ToNot(HaveOccurred())
	defer f.Close()
	want = regexp.QuoteMeta(want)
	Eventually(gbytes.BufferReader(f)).WithTimeout(timeout).Should(gbytes.Say(want))
}

// WriteTempFile writes content into name under a temp folder.
func WriteTempFile(name string, content string) (path string) {
	tempDir := GinkgoT().TempDir()
	path = filepath.Join(tempDir, name)
	err := os.WriteFile(path, []byte(content), 0666)
	Expect(err).ToNot(HaveOccurred())
	return path
}

func copyFile(srcFile, dstFile string) error {
	to, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer to.Close()

	from, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer from.Close()

	_, err = io.Copy(to, from)
	return err
}
