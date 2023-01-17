//go:build windows

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

package fileref

import (
	"testing"
)

func Test_doParse(t *testing.T) {
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
		{"windows file name2 and default type", args{`z:b`, "c"}, `z`, "b"},
		{"windows file name and media type", args{`a:\b:c`, "d"}, `a:\b`, "c"},
		{"windows file name and empty media type", args{`a:\b:`, "c"}, `a:\b`, ""},
		{"numeric file name and media type", args{`1:\a`, "b"}, `1`, `\a`},
		{"non-windows file name and media type", args{`ab:\c`, ""}, `ab`, `\c`},
		{"non-windows file name and media type, default type ignored", args{`1:\a`, "b"}, `1`, `\a`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFilePath, gotMediatype := doParse(tt.args.reference, tt.args.mediaType)
			if gotFilePath != tt.wantFilePath {
				t.Errorf("doParse() gotFilePath = %v, want %v", gotFilePath, tt.wantFilePath)
			}
			if gotMediatype != tt.wantMediatype {
				t.Errorf("doParse() gotMediatype = %v, want %v", gotMediatype, tt.wantMediatype)
			}
		})
	}
}

func TestParse(t *testing.T) {
	type args struct {
		reference string
		mediaType string
	}
	tests := []struct {
		name          string
		args          args
		wantFilePath  string
		wantMediatype string
		wantErr       bool
	}{
		{"valid file name", args{`c:\some-folder\test`, ""}, `c:\some-folder\test`, "", false},
		{"valid file name and media type", args{`c:\some-folder\test`, "type"}, `c:\some-folder\test`, "type", false},
		{"no input", args{"", ""}, "", "", true},
		{"empty file name", args{":", ""}, "", "", true},
		{"reserved character1 in file name", args{"<", "a"}, "", "", true},
		{"reserved character2 in file name", args{">", "a"}, "", "", true},
		{"reserved character3 in file name", args{"*", "a"}, "", "", true},
		{"reserved character4 in file name", args{`"`, "a"}, "", "", true},
		{"reserved character5 in file name", args{"|", "a"}, "", "", true},
		{"reserved character6 in file name", args{"?", "a"}, "", "", true},
		{"empty file name, with media type", args{":", "a"}, "", "", true},
		{"reserved character1 in file name, with media type", args{"<:", "a"}, "", "", true},
		{"reserved character2 in file name, with media type", args{">:", "a"}, "", "", true},
		{"reserved character3 in file name, with media type", args{"*:", "a"}, "", "", true},
		{"reserved character4 in file name, with media type", args{`":`, "a"}, "", "", true},
		{"reserved character5 in file name, with media type", args{"|:", "a"}, "", "", true},
		{"reserved character6 in file name, with media type", args{"?:", "a"}, "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFilePath, gotMediatype, err := Parse(tt.args.reference, tt.args.mediaType)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotFilePath != tt.wantFilePath {
				t.Errorf("Parse() gotFilePath = %v, want %v", gotFilePath, tt.wantFilePath)
			}
			if gotMediatype != tt.wantMediatype {
				t.Errorf("Parse() gotMediatype = %v, want %v", gotMediatype, tt.wantMediatype)
			}
		})
	}
}
