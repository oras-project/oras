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
)

func TestPacker_FlagInit(t *testing.T) {
	// flag init
}

func TestPacker_LoadManifestAnnotations(t *testing.T) {
	// when --manifest--anotation and --manifest-annotation-file are specified exit with error.
	testContent := `{
		"$config": {
		  "hello": "world"
		},
		"$manifest": {
		  "foo": "bar"
		},
		"cake.txt": {
		  "fun": "more cream"
		}
	  }`
	testFile := filepath.Join(t.TempDir(), "testAnnotationFile")
	os.WriteFile(testFile, []byte(testContent), fs.ModePerm)
	opts := Packer{
		AnnotationFilePath:  testFile,
		ManifestAnnotations: []string{"Key=Val"},
	}
	if _, err := opts.LoadManifestAnnotations(); !errors.Is(err, errAnnotationConflict) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPacker_decodeJSON(t *testing.T) {
	testContent := `{
		"$config": {
		  "hello": "world"
		},
		"$manifest": {
		  "foo": "bar"
		},
		"cake.txt": {
		  "fun": "more cream"
		}
	  }`
	testFile := filepath.Join(t.TempDir(), "testAnnotationFile")
	os.WriteFile(testFile, []byte(testContent), fs.ModePerm)
	opts := Packer{
		AnnotationFilePath: testFile,
	}
	annotations := make(map[string]map[string]string)
	err := decodeJSON(opts.AnnotationFilePath, &annotations)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(annotations, map[string]map[string]string{
		"$config": {
			"hello": "world",
		},
		"$manifest": {
			"foo": "bar",
		},
		"cake.txt": {
			"fun": "more cream",
		},
	}) {
		t.Fatalf("unexpected error: %v", errors.New("content not match"))
	}
}

func TestPacker_parseAnnotationFlags(t *testing.T) {
	// Item do not contains '='
	invalidAnnotations0 := []string{
		"Key",
	}
	// Duplication Key
	invalidAnnotations1 := []string{
		"Key=0",
		"Key=1",
	}
	// Valid Annotations
	manifestAnnotations := []string{
		"Key0=",                // 1. Item not contains 'val'
		"Key1=Val",             // 2. Normal Item
		"Key2=${env:USERNAME}", // 3. Item contains variable eg. "${env:USERNAME}"
		" Key3 = Val ",         // 4. Item trim conversion
	}
	annotations := map[string]map[string]string{}
	if err := parseAnnotationFlags(invalidAnnotations0, annotations); !errors.Is(err, errAnnotationFormat) {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := parseAnnotationFlags(invalidAnnotations1, annotations); !errors.Is(err, errAnnotationDuplication) {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := parseAnnotationFlags(manifestAnnotations, annotations); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := annotations["$manifest"]; !ok {
		t.Fatalf("unexpected error: failed when looking for '$manifest' in annotations")
	}
	if !reflect.DeepEqual(annotations,
		map[string]map[string]string{"$manifest": {
			"Key0": "",
			"Key1": "Val",
			"Key2": "${env:USERNAME}",
			"Key3": "Val",
		},
		}) {
		t.Fatalf("unexpected error: %v", errors.New("content not match"))
	}
}
