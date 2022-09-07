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
	"os"
	"path/filepath"
)

func ReadFullFile(path string) ([]byte, error) {
	fp, err := os.OpenFile(path, os.O_RDONLY, 0666)
	if err != nil {
		return nil, err
	}

	defer fp.Close()
	return io.ReadAll(fp)
}

func CopyTestData(fileNames []string, dstRoot string) error {
	for _, name := range fileNames {
		// make sure all parents are created
		if err := os.MkdirAll(filepath.Join(dstRoot, filepath.Dir(name)), 0700); err != nil {
			return err
		}

		if err := copyFile(filepath.Join(imageDir, name), filepath.Join(dstRoot, name)); err != nil {
			return err
		}
	}
	return nil
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
	if err != nil {
		return err
	}

	return nil
}
