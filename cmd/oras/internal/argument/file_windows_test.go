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

package argument

import "testing"

func TestParseFileReference(t *testing.T) {
	type args struct {
		reference string
		mediaType string
	}
	tests := []struct {
		name          string
		args          args
		wantFilePath  string
		wantMediatype string
	}{
		// {"file name and media type in reference", args{"a:b", "c"}, "a", "b"},
		// {"media type in reference", args{":b", "c"}, "", "b"},
		// {"file name and empty media type in reference", args{"a:", "c"}, "a", ""},
		// {"file name in reference", args{"a", "c"}, "a", "c"},
		// {"file name in reference, no default", args{"a:", ""}, "a", ""},
		// {"file name in reference with default media type", args{`a:\b`, "d"}, `a:\b`, "c"},
		{"file name and media type in reference", args{"a:b:c", "d"}, "a:b", "c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFilePath, gotMediatype := ParseFileReference(tt.args.reference, tt.args.mediaType)
			if gotFilePath != tt.wantFilePath {
				t.Errorf("ParseFileReference() gotFilePath = %v, want %v", gotFilePath, tt.wantFilePath)
			}
			if gotMediatype != tt.wantMediatype {
				t.Errorf("ParseFileReference() gotMediatype = %v, want %v", gotMediatype, tt.wantMediatype)
			}
		})
	}
}
