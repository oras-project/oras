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

package index

import (
	"reflect"
	"testing"
)

func Test_parseAnnotations(t *testing.T) {
	tests := []struct {
		name            string
		input           []string
		annotations     map[string]string
		wantErr         bool
		wantAnnotations map[string]string
	}{
		{
			name:            "valid input",
			input:           []string{"a=b", "c=d", "e=f"},
			wantErr:         false,
			wantAnnotations: map[string]string{"a": "b", "c": "d", "e": "f"},
		},
		{
			name:            "invalid input",
			input:           []string{"a=b", "c:d", "e=f"},
			wantErr:         true,
			wantAnnotations: nil,
		},
		{
			name:            "duplicate key",
			input:           []string{"a=b", "c=d", "a=e"},
			wantErr:         true,
			wantAnnotations: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotations, err := parseAnnotations(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAnnotations() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(annotations, tt.wantAnnotations) {
				t.Errorf("parseAnnotations() annotations = %v, want %v", tt.annotations, tt.wantAnnotations)
			}
		})
	}
}
