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

package file

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
		{"file name and media type", args{"az:b", ""}, "az", "b"},
		{"file name and empty media type", args{"az:", ""}, "az", ""},
		{"file name and default media type", args{"az", "c"}, "az", "c"},
		{"file name and media type, default type ignored", args{"az:b", "c"}, "az", "b"},
		{"file name and empty media type, default type ignored", args{"az:", "c"}, "az", ""},

		{"empty file name and media type", args{":a", "b"}, "", "a"},
		{"empty file name and empty media type", args{":", "a"}, "", ""},
		{"empty name and default media type", args{"", "a"}, "", "a"},

		{"colon file name and media type", args{"az:b:c", "d"}, "az:b", "c"},
		{"colon file name and empty media type", args{"az:b:", "c"}, "az:b", ""},
		{"colon-prefix file name and media type", args{":az:b:c", "d"}, ":az:b", "c"},

		{"pure colon file name and media type", args{"::a", "b"}, ":", "a"},
		{"pure colon file name and empty media type", args{"::", "a"}, ":", ""},

		{"windows file name1 and default type", args{`a:\b`, "c"}, `a:\b`, "c"},
		{"windows file name2 and default type", args{`z:b`, "c"}, `Z:\b`, "c"},
		{"windows file name and media type", args{`a:\b:c`, "d"}, `a:\b`, "c"},
		{"windows file name and empty media type", args{`a:\b:`, "c"}, `a:\b`, ""},
		{"windows file name and empty media type", args{`a:\b:`, "c"}, `a:\b`, ""},
		{"numeric file name and media type", args{`1:\a`, "b"}, `1`, `\a`},
		{"non-windows file name and media type", args{`ab:\c`, ""}, `ab`, `\c`},
		{"non-windows file name and media type, default type ignored", args{`1:\a`, "b"}, `1`, `\a`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFilePath, gotMediatype := Parse(tt.args.reference, tt.args.mediaType)
			if gotFilePath != tt.wantFilePath {
				t.Errorf("ParseFileReference() gotFilePath = %v, want %v", gotFilePath, tt.wantFilePath)
			}
			if gotMediatype != tt.wantMediatype {
				t.Errorf("ParseFileReference() gotMediatype = %v, want %v", gotMediatype, tt.wantMediatype)
			}
		})
	}
}
