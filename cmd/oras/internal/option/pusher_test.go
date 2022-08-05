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
	manifestAnnotations := []string{"testkey=testVal"}
	annotations := map[string]map[string]string{}
	if err := getAnnotationsMap(manifestAnnotations, annotations); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assert.Assert(t, cmp.Contains(annotations, "$manifest"))
}
