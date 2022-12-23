//go:build !windows

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

import "testing"

func Test_ParseFileReference(t *testing.T) {
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
		{"colon file name and media type", args{"az:b:c", "d"}, "az:b", "c"},
		{"colon file name and empty media type", args{"az:b:", "c"}, "az:b", ""},
		{"colon-prefix file name and media type", args{":az:b:c", "d"}, ":az:b", "c"},

		{"pure colon file name and media type", args{"::a", "b"}, ":", "a"},
		{"pure colon file name and empty media type", args{"::", "a"}, ":", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFilePath, gotMediatype, _ := Parse(tt.args.reference, tt.args.mediaType)
			if gotFilePath != tt.wantFilePath {
				t.Errorf("Parse() gotFilePath = %v, want %v", gotFilePath, tt.wantFilePath)
			}
			if gotMediatype != tt.wantMediatype {
				t.Errorf("Parse() gotMediatype = %v, want %v", gotMediatype, tt.wantMediatype)
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

		{"no input", args{"", ""}, "", "", true},
		{"empty file name and media type", args{":", ""}, "", "", true},
		{"empty file name with media type", args{":a", "b"}, "", "", true},
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
