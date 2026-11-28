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

import "testing"

func TestValidateExtraTags(t *testing.T) {
	tests := []struct {
		name    string
		tags    []string
		wantErr bool
	}{
		{
			name: "no extra tags",
		},
		{
			name: "valid tags",
			tags: []string{"v2", "latest"},
		},
		{
			name:    "empty tag",
			tags:    []string{""},
			wantErr: true,
		},
		{
			name:    "empty tag between valid tags",
			tags:    []string{"v2", "", "v3"},
			wantErr: true,
		},
		{
			name:    "invalid tag",
			tags:    []string{"invalid@@tag"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExtraTags(tt.tags)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateExtraTags(%q) error = %v, wantErr %t",
					tt.tags, err, tt.wantErr)
			}
		})
	}
}

func TestTarget_ExtraTagReference(t *testing.T) {
	tests := []struct {
		name   string
		target Target
		want   string
	}{
		{
			name:   "remote target",
			target: Target{Type: TargetTypeRemote, Reference: "v1"},
			want:   "v2",
		},
		{
			name:   "OCI layout path with a repository-qualified reference",
			target: Target{Type: TargetTypeOCILayout, Reference: "localhost:5000/repo:v1"},
			want:   "localhost:5000/repo:v2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.target.ExtraTagReference("v2"); got != tt.want {
				t.Errorf("ExtraTagReference() = %q, want %q", got, tt.want)
			}
		})
	}
}
