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

package option

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/pflag"
)

const testContent = `{"$config":{"hello":"world"},"$manifest":{"foo":"bar"},"cake.txt":{"fun":"more cream"}}`

var expectedResult = map[string]map[string]string{"$config": {"hello": "world"}, "$manifest": {"foo": "bar"}, "cake.txt": {"fun": "more cream"}}

func TestPacker_FlagInit(t *testing.T) {
	var test struct {
		Packer
	}
	ApplyFlags(&test, pflag.NewFlagSet("oras-test", pflag.ExitOnError))
}

func TestPacker_parseAnnotations_err(t *testing.T) {
	opts := Packer{
		Annotation: Annotation{
			ManifestAnnotations: []string{"Key=Val"},
		},
		AnnotationFilePath: "this is not a file", // testFile,
	}
	if err := opts.parseAnnotations(nil); !errors.Is(err, errAnnotationConflict) {
		t.Fatalf("unexpected error: %v", err)
	}

	opts = Packer{
		AnnotationFilePath: "this is not a file", // testFile,
	}
	if err := opts.parseAnnotations(nil); err == nil {
		t.Fatalf("unexpected error: %v", err)
	}

	opts = Packer{
		Annotation: Annotation{
			ManifestAnnotations: []string{"KeyVal"},
		},
	}
	if err := opts.parseAnnotations(nil); !errors.Is(err, errAnnotationFormat) {
		t.Fatalf("unexpected error: %v", err)
	}

	opts = Packer{
		Annotation: Annotation{
			ManifestAnnotations: []string{"Key=Val1", "Key=Val2"},
		},
	}
	if err := opts.parseAnnotations(nil); !errors.Is(err, errAnnotationDuplication) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPacker_parseAnnotations_annotationFile(t *testing.T) {
	testFile := filepath.Join(t.TempDir(), "testAnnotationFile")
	err := os.WriteFile(testFile, []byte(testContent), fs.ModePerm)
	if err != nil {
		t.Fatalf("Error writing %s: %v", testFile, err)
	}
	opts := Packer{
		AnnotationFilePath: testFile,
	}

	err = opts.parseAnnotations(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(opts.Annotations, expectedResult) {
		t.Fatalf("unexpected error: %v", opts.Annotations)
	}
}

func TestPacker_parseAnnotations_annotationFlag(t *testing.T) {
	// Item do not contains '='
	invalidFlag0 := []string{
		"Key",
	}
	opts := Packer{
		Annotation: Annotation{
			ManifestAnnotations: invalidFlag0,
		},
	}
	err := opts.parseAnnotations(nil)
	if !errors.Is(err, errAnnotationFormat) {
		t.Fatalf("unexpected error: %v", err)
	}

	// Duplication Key
	invalidFlag1 := []string{
		"Key=0",
		"Key=1",
	}
	opts = Packer{
		Annotation: Annotation{
			ManifestAnnotations: invalidFlag1,
		},
	}
	err = opts.parseAnnotations(nil)
	if !errors.Is(err, errAnnotationDuplication) {
		t.Fatalf("unexpected error: %v", err)
	}

	// Valid Annotations
	validFlag := []string{
		"Key0=",                // 1. Item not contains 'val'
		"Key1=Val",             // 2. Normal Item
		"Key2=${env:USERNAME}", // 3. Item contains variable eg. "${env:USERNAME}"
	}
	opts = Packer{
		Annotation: Annotation{
			ManifestAnnotations: validFlag,
		},
	}
	err = opts.parseAnnotations(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := opts.Annotations["$manifest"]; !ok {
		t.Fatalf("unexpected error: failed when looking for '$manifest' in annotations")
	}
	if !reflect.DeepEqual(opts.Annotations,
		map[string]map[string]string{
			"$manifest": {
				"Key0": "",
				"Key1": "Val",
				"Key2": "${env:USERNAME}",
			},
		}) {
		t.Fatalf("unexpected error: %v", errors.New("content not match"))
	}
}

func givenTestFile(t *testing.T, data string) (path string) {
	tempDir := t.TempDir()
	fileName := "test.txt"
	path = filepath.Join(tempDir, fileName)
	content := []byte(data)
	if err := os.WriteFile(path, content, 0444); err != nil {
		t.Fatal("error calling WriteFile(), error =", err)
	}
	return path
}

func TestPacker_decodeJSON(t *testing.T) {
	var annotation Annotation

	path := "nonexistent-file.json"
	err := decodeJSON(path, &annotation.Annotations)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("unexpected error: %v", err)
	}

	path = givenTestFile(t, "bogus data")
	err = decodeJSON(path, &annotation.Annotations)
	if err == nil || err.Error() != "invalid character 'b' looking for beginning of value" {
		t.Fatalf("unexpected error: %v", err)
	}

	path = givenTestFile(t, "{\"annotations\":{\"org.opencontainers.image.ref.name\":\"ghcr.io/stefanprodan/podinfo:6.8.0\"}}")
	err = decodeJSON(path, &annotation.Annotations)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
