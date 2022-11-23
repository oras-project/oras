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
	"time"

	. "github.com/onsi/gomega"
)

var testFileRoot string

// CopyTestData copies test data into the temp test folder.
func CopyTestData(dstRoot string) error {
	return filepath.WalkDir(testFileRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// ignore folder
			return nil
		}

		relPath, err := filepath.Rel(testFileRoot, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dstRoot, relPath)
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
	Expect(filepath).Should(BeAnExistingFile())
	f, err := os.Open(filepath)
	Expect(err).ShouldNot(HaveOccurred())
	defer f.Close()
	content, err := os.ReadFile(filepath)
	Expect(err).ShouldNot(HaveOccurred())
	Eventually(string(content)).WithTimeout(timeout).Should(Equal(want))
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
