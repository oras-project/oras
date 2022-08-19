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
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"oras.land/oras/cmd/oras/internal/file"
)

const manifest = `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"mediaType":"application/vnd.unknown.config.v1+json","digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2},"layers":[{"mediaType":"application/vnd.oci.image.layer.v1.tar","digest":"sha256:5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03","size":6,"annotations":{"org.opencontainers.image.title":"hello.txt"}}]}`

func TestFile_ParseMediaType(t *testing.T) {
	// generate test content
	tempDir := t.TempDir()
	content := []byte(manifest)
	fileName := "manifest.json"
	path := filepath.Join(tempDir, fileName)
	if err := os.WriteFile(path, content, 0444); err != nil {
		t.Fatal("error calling WriteFile(), error =", err)
	}

	// test ParseMediaType
	want := "application/vnd.oci.image.manifest.v1+json"
	got, err := file.ParseMediaType(path)
	if err != nil {
		t.Fatal("ParseMediaType() error=", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseMediaType() = %v, want %v", got, want)
	}
}

func TestFile_ParseMediaType_invalidContent_NotAJson(t *testing.T) {
	// generate test content
	tempDir := t.TempDir()
	content := []byte("manifest")
	fileName := "manifest.txt"
	path := filepath.Join(tempDir, fileName)
	if err := os.WriteFile(path, content, 0444); err != nil {
		t.Fatal("error calling WriteFile(), error =", err)
	}

	// test ParseMediaType
	_, err := file.ParseMediaType(path)
	expected := "invalid character 'm' looking for beginning of value"
	if err.Error() != expected {
		t.Fatalf("ParseMediaType() error = %v, wantErr %v", err, expected)
	}
}

func TestFile_ParseMediaType_invalidContent_MissingMediaType(t *testing.T) {
	// generate test content
	tempDir := t.TempDir()
	content := []byte(`{"schemaVersion":2}`)
	fileName := "manifest.json"
	path := filepath.Join(tempDir, fileName)
	if err := os.WriteFile(path, content, 0444); err != nil {
		t.Fatal("error calling WriteFile(), error =", err)
	}

	// test ParseMediaType
	_, err := file.ParseMediaType(path)
	expected := "media type is not recognized"
	if err.Error() != expected {
		t.Fatalf("ParseMediaType() error = %v, wantErr %v", err, expected)
	}
}
