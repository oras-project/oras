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
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestPusher_LoadManifestAnnotations(t *testing.T) {
	opts := Pusher{
		AnnotationsFilePath: "",
		ManifestAnnotations: []string{"testKey=testVal"},
	}
	annotation, err := opts.LoadManifestAnnotations()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assert.Assert(t, cmp.Contains(annotation, "$manifest"))
}

func TestPusher_getAnnotationsMap(t *testing.T) {
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
	if err := getAnnotationsMap(invalidAnnotations0, annotations); err != nil {
		assert.Error(t, err, "annotation MUST be a key-value pair")
	}
	if err := getAnnotationsMap(invalidAnnotations1, annotations); err != nil {
		assert.Error(t, err, "annotation key conflict")
	}
	if err := getAnnotationsMap(manifestAnnotations, annotations); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := annotations["$manifest"]; !ok {
		t.Fatalf("unexpected error: failed when looking for '$manifest' in annotations")
	}
	assert.DeepEqual(t, annotations, map[string]map[string]string{"$manifest": {
		"Key0": "",
		"Key1": "Val",
		"Key2": "${env:USERNAME}",
		"Key3": "Val",
	}})
}
