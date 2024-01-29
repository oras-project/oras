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

package display

import (
	"fmt"
	"os"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// FileHandler handles file store operations.
type FileHandler struct {
	needTextOutput bool
	verbose        bool
}

// NewFileHandler creates a new handler for file store operations.
func NewFileHandler(template string, tty *os.File, verbose bool) *FileHandler {
	fh := &FileHandler{
		needTextOutput: NeedTextOutput(template, tty),
		verbose:        verbose,
	}
	return fh
}

// PreAdd is called before adding a new file to file store.
func (fh *FileHandler) PreAdd(name string) {
	if fh.needTextOutput && fh.verbose {
		_, _ = fmt.Println("Preparing", name)
	}
}

// PostAdd is called after adding all files to file store.
func (fh *FileHandler) PostAdd(files []ocispec.Descriptor) {
	if fh.needTextOutput && len(files) == 0 {
		_, _ = fmt.Println("Uploading empty artifact")
	}
}
