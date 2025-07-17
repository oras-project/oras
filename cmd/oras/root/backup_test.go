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

package root

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseArtifactRefs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantRepo string
		wantRefs []string
		wantErr  bool
	}{
		// Basic test cases
		{
			name:     "valid reference with tag",
			input:    "localhost:5000/repo:v1",
			wantRepo: "localhost:5000/repo",
			wantRefs: []string{"v1"},
			wantErr:  false,
		},
		{
			name:     "valid reference with multiple tags",
			input:    "localhost:5000/repo:v1,v2",
			wantRepo: "localhost:5000/repo",
			wantRefs: []string{"v1", "v2"},
			wantErr:  false,
		},
		{
			name:     "complex repository with multiple slashes",
			input:    "localhost:5000/org/team/project:v1,v2",
			wantRepo: "localhost:5000/org/team/project",
			wantRefs: []string{"v1", "v2"},
			wantErr:  false,
		},
		{
			name:     "reference with empty tag",
			input:    "localhost:5000/repo:",
			wantRepo: "localhost:5000/repo",
			wantRefs: []string{},
			wantErr:  false,
		},
		{
			name:     "valid reference with empty first tag",
			input:    "localhost:5000/repo:,v1",
			wantRepo: "localhost:5000/repo",
			wantRefs: []string{"v1"},
			wantErr:  false,
		},
		{
			name:     "valid reference without tag",
			input:    "localhost:5000/repo",
			wantRepo: "localhost:5000/repo",
			wantRefs: []string{},
			wantErr:  false,
		},
		{
			name:     "valid reference with tag and extra refs",
			input:    "localhost:5000/repo:v1,v2,v3",
			wantRepo: "localhost:5000/repo",
			wantRefs: []string{"v1", "v2", "v3"},
			wantErr:  false,
		},
		{
			name:     "invalid reference with digest",
			input:    "localhost:5000/repo@sha256:a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447",
			wantRepo: "",
			wantRefs: nil,
			wantErr:  true,
		},
		{
			name:     "invalid reference with digest and extra ref",
			input:    "localhost:5000/repo@sha256:a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447,v1",
			wantRepo: "",
			wantRefs: nil,
			wantErr:  true,
		},

		// Edge cases
		{
			name:     "empty reference",
			input:    "",
			wantRepo: "",
			wantRefs: nil,
			wantErr:  true,
		},
		{
			name:     "multiple empty tags",
			input:    "localhost:5000/repo:,,",
			wantRepo: "localhost:5000/repo",
			wantRefs: []string{},
			wantErr:  false,
		},
		{
			name:     "invalid tag format",
			input:    "localhost:5000/repo:valid,invalid@tag",
			wantRepo: "",
			wantRefs: nil,
			wantErr:  true,
		},
		{
			name:     "valid tag with special characters",
			input:    "localhost:5000/repo:v1.0-beta_1",
			wantRepo: "localhost:5000/repo",
			wantRefs: []string{"v1.0-beta_1"},
			wantErr:  false,
		},
		{
			name:     "invalid repository format",
			input:    "localhost:5000/repo space:v1",
			wantRepo: "",
			wantRefs: nil,
			wantErr:  true,
		},
		{
			name:     "tag exceeding max length",
			input:    "localhost:5000/repo:" + strings.Repeat("a", 129),
			wantRepo: "",
			wantRefs: nil,
			wantErr:  true,
		},
		{
			name:     "invalid tag starting with non-word character",
			input:    "localhost:5000/repo:.invalid",
			wantRepo: "",
			wantRefs: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository, references, err := parseArtifactRefs(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if repository != tt.wantRepo {
				t.Errorf("expected repository %q, got %q", tt.wantRepo, repository)
			}
			if !reflect.DeepEqual(references, tt.wantRefs) {
				t.Errorf("expected references %v, got %v", tt.wantRefs, references)
			}
		})
	}
}
