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

package model

import (
	"slices"
	"testing"
)

func TestTagged_AddTag(t *testing.T) {
	var tagged Tagged
	for _, tag := range []string{"v1.0.0", "latest"} {
		tagged.AddTag(tag)
	}

	got := tagged.Tags()
	for _, want := range []string{"v1.0.0", "latest"} {
		if !slices.Contains(got, want) {
			t.Errorf("Tags() = %v, want it to contain %q", got, want)
		}
	}
}

// Tags sorts, so callers see a stable order regardless of the order the tags
// were recorded in. push relies on this for referenceAsTags.
func TestTagged_Tags_sorted(t *testing.T) {
	var tagged Tagged
	for _, tag := range []string{"v2.0.0", "latest", "v1.0.0"} {
		tagged.AddTag(tag)
	}

	got := tagged.Tags()
	want := []string{"latest", "v1.0.0", "v2.0.0"}
	if !slices.Equal(got, want) {
		t.Errorf("Tags() = %v, want %v", got, want)
	}
}

func TestTagged_Tags_empty(t *testing.T) {
	var tagged Tagged
	if got := tagged.Tags(); len(got) != 0 {
		t.Errorf("Tags() = %v, want empty", got)
	}
}
