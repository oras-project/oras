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

package option

import (
	"testing"
)

func Test_parseOCILayoutReference_windows(t *testing.T) {
	opts := Target{
		RawReference: `C:\some-folder:tag`,
		IsOCILayout:  true,
	}
	tests := []struct {
		name    string
		want    string
		want1   string
		wantErr bool
	}{
		{"path and tag", `C:\some-folder`, "tag", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := opts.parseOCILayoutReference()
			if (err != nil) != tt.wantErr {
				t.Errorf("parseOCILayoutReference() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if opts.Path != tt.want {
				t.Errorf("parseOCILayoutReference() opts.Path = %v, want %v", opts.Path, tt.want)
			}
			if opts.Reference != tt.want1 {
				t.Errorf("parseOCILayoutReference() opts.Reference = %v, want %v", opts.Reference, tt.want1)
			}
		})
	}
}
