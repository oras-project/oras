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
	"strings"
	"testing"
)

// Test_tagCmd_invalidTag checks that a malformed destination tag is rejected
// before the target is touched. An OCI image layout is used as the target so
// that the check runs without a registry.
func Test_tagCmd_invalidTag(t *testing.T) {
	tests := []struct {
		name    string
		tag     string
		wantErr string
	}{
		{"empty tag", "", "empty extra tag"},
		{"tag with a space", "invalid tag", "invalid extra tag"},
		{"tag with a leading period", ".invalid", "invalid extra tag"},
		{"tag with a slash", "invalid/tag", "invalid extra tag"},
		{"tag that is too long", strings.Repeat("v", 129), "invalid extra tag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tagCmd()
			cmd.SetArgs([]string{t.TempDir() + ":v1", "--oci-layout", tt.tag})
			err := cmd.Execute()
			if err == nil {
				t.Fatalf("tagCmd() error = nil, want %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("tagCmd() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}
