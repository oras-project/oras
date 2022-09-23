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

package file_test

import (
	"reflect"
	"testing"

	"oras.land/oras-go/v2"
	"oras.land/oras/internal/file"
)

func TestFile_ParseFileReference(t *testing.T) {
	fileRef := "demo/hi.txt:application/vnd.me.hi"

	wantFilePath := "demo/hi.txt"
	wantMediaType := "application/vnd.me.hi"

	// test ParseFileReference
	gotFilePath, gotMediaType := file.ParseFileReference(fileRef, "")
	if !reflect.DeepEqual(gotFilePath, wantFilePath) {
		t.Errorf("ParseFileReference() = %v, want %v", gotFilePath, wantFilePath)
	}
	if !reflect.DeepEqual(gotMediaType, wantMediaType) {
		t.Errorf("ParseFileReference() = %v, want %v", gotMediaType, wantMediaType)
	}
}

func TestFile_ParseFileReference_withMediaTypeInput(t *testing.T) {
	fileRef := "demo/config.json"

	wantFilePath := "demo/config.json"
	wantMediaType := oras.MediaTypeUnknownConfig

	// test ParseFileReference
	gotFilePath, gotMediaType := file.ParseFileReference(fileRef, oras.MediaTypeUnknownConfig)
	if !reflect.DeepEqual(gotFilePath, wantFilePath) {
		t.Errorf("ParseFileReference() = %v, want %v", gotFilePath, wantFilePath)
	}
	if !reflect.DeepEqual(gotMediaType, wantMediaType) {
		t.Errorf("ParseFileReference() = %v, want %v", gotMediaType, wantMediaType)
	}
}

func TestFile_ParseFileReference_includingWindowsDisk(t *testing.T) {
	fileRef := `C:\demo\hi.txt:application/vnd.me.hi`

	wantFilePath := `C:\demo\hi.txt`
	wantMediaType := "application/vnd.me.hi"

	// test ParseFileReference
	gotFilePath, gotMediaType := file.ParseFileReference(fileRef, oras.MediaTypeUnknownConfig)
	if !reflect.DeepEqual(gotFilePath, wantFilePath) {
		t.Errorf("ParseFileReference() = %v, want %v", gotFilePath, wantFilePath)
	}
	if !reflect.DeepEqual(gotMediaType, wantMediaType) {
		t.Errorf("ParseFileReference() = %v, want %v", gotMediaType, wantMediaType)
	}
}
