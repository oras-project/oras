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

func TestPacker_LoadManifestAnnotations_err(t *testing.T) {
	opts := Packer{
		AnnotationFilePath:  "this is not a file", // testFile,
		ManifestAnnotations: []string{"Key=Val"},
	}
	if _, err := opts.LoadManifestAnnotations(); !errors.Is(err, errAnnotationConflict) {
		t.Fatalf("unexpected error: %v", err)
	}

	opts = Packer{
		AnnotationFilePath: "this is not a file", // testFile,
	}
	if _, err := opts.LoadManifestAnnotations(); err == nil {
		t.Fatalf("unexpected error: %v", err)
	}

	opts = Packer{
		ManifestAnnotations: []string{"KeyVal"},
	}
	if _, err := opts.LoadManifestAnnotations(); !errors.Is(err, errAnnotationFormat) {
		t.Fatalf("unexpected error: %v", err)
	}

	opts = Packer{
		ManifestAnnotations: []string{"Key=Val1", "Key=Val2"},
	}
	if _, err := opts.LoadManifestAnnotations(); !errors.Is(err, errAnnotationDuplication) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPacker_LoadManifestAnnotations_annotationFile(t *testing.T) {
	testFile := filepath.Join(t.TempDir(), "testAnnotationFile")
	err := os.WriteFile(testFile, []byte(testContent), fs.ModePerm)
	if err != nil {
		t.Fatalf("Error writing %s: %v", testFile, err)
	}
	opts := Packer{AnnotationFilePath: testFile}

	anno, err := opts.LoadManifestAnnotations()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(anno, expectedResult) {
		t.Fatalf("unexpected error: %v", anno)
	}
}

func TestPacker_LoadManifestAnnotations_annotationFlag(t *testing.T) {
	// Item do not contains '='
	invalidFlag0 := []string{
		"Key",
	}
	var annotations map[string]map[string]string
	opts := Packer{ManifestAnnotations: invalidFlag0}
	_, err := opts.LoadManifestAnnotations()
	if !errors.Is(err, errAnnotationFormat) {
		t.Fatalf("unexpected error: %v", err)
	}

	// Duplication Key
	invalidFlag1 := []string{
		"Key=0",
		"Key=1",
	}
	opts = Packer{ManifestAnnotations: invalidFlag1}
	_, err = opts.LoadManifestAnnotations()
	if !errors.Is(err, errAnnotationDuplication) {
		t.Fatalf("unexpected error: %v", err)
	}

	// Valid Annotations
	validFlag := []string{
		"Key0=",                // 1. Item not contains 'val'
		"Key1=Val",             // 2. Normal Item
		"Key2=${env:USERNAME}", // 3. Item contains variable eg. "${env:USERNAME}"
	}
	opts = Packer{ManifestAnnotations: validFlag}
	annotations, err = opts.LoadManifestAnnotations()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := annotations["$manifest"]; !ok {
		t.Fatalf("unexpected error: failed when looking for '$manifest' in annotations")
	}
	if !reflect.DeepEqual(annotations,
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
