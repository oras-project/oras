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

func TestExactly(t *testing.T) {
	tests := []struct {
		name    string
		cnt     int
		args    []string
		wantOK  bool
		wantMsg string
	}{
		{"none required, none given", 0, nil, true, "exactly 0 arguments"},
		{"none required, one given", 0, []string{"a"}, false, "exactly 0 arguments"},
		{"one required, none given", 1, nil, false, "exactly 1 argument"},
		{"one required, one given", 1, []string{"a"}, true, "exactly 1 argument"},
		{"one required, two given", 1, []string{"a", "b"}, false, "exactly 1 argument"},
		{"two required, one given", 2, []string{"a"}, false, "exactly 2 arguments"},
		{"two required, two given", 2, []string{"a", "b"}, true, "exactly 2 arguments"},
		{"two required, three given", 2, []string{"a", "b", "c"}, false, "exactly 2 arguments"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOK, gotMsg := Exactly(tt.cnt)(tt.args)
			if gotOK != tt.wantOK {
				t.Errorf("Exactly(%d)(%v) ok = %v, want %v", tt.cnt, tt.args, gotOK, tt.wantOK)
			}
			if gotMsg != tt.wantMsg {
				t.Errorf("Exactly(%d)(%v) msg = %q, want %q", tt.cnt, tt.args, gotMsg, tt.wantMsg)
			}
		})
	}
}

func TestAtLeast(t *testing.T) {
	tests := []struct {
		name    string
		cnt     int
		args    []string
		wantOK  bool
		wantMsg string
	}{
		{"none required, none given", 0, nil, true, "at least 0 arguments"},
		{"none required, one given", 0, []string{"a"}, true, "at least 0 arguments"},
		{"one required, none given", 1, nil, false, "at least 1 argument"},
		{"one required, one given", 1, []string{"a"}, true, "at least 1 argument"},
		{"one required, two given", 1, []string{"a", "b"}, true, "at least 1 argument"},
		{"two required, one given", 2, []string{"a"}, false, "at least 2 arguments"},
		{"two required, two given", 2, []string{"a", "b"}, true, "at least 2 arguments"},
		{"two required, three given", 2, []string{"a", "b", "c"}, true, "at least 2 arguments"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOK, gotMsg := AtLeast(tt.cnt)(tt.args)
			if gotOK != tt.wantOK {
				t.Errorf("AtLeast(%d)(%v) ok = %v, want %v", tt.cnt, tt.args, gotOK, tt.wantOK)
			}
			if gotMsg != tt.wantMsg {
				t.Errorf("AtLeast(%d)(%v) msg = %q, want %q", tt.cnt, tt.args, gotMsg, tt.wantMsg)
			}
		})
	}
}
