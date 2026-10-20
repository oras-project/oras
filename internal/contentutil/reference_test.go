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

package contentutil

import "testing"

func TestIsDigest(t *testing.T) {
	tests := []struct {
		tagOrDigest string
		want        bool
	}{
		{"sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", true},
		{"sha512:cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e", true},
		{"v1.0.0", false},
		{"latest", false},
		{"", false},
		// A digest must have the length its algorithm defines.
		{"sha256:e3b0c442", false},
		// The encoded part of a sha256 digest must be lower-case hex.
		{"sha256:E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855", false},
		{"sha256:", false},
	}
	for _, tt := range tests {
		if got := IsDigest(tt.tagOrDigest); got != tt.want {
			t.Errorf("IsDigest(%q) = %v, want %v", tt.tagOrDigest, got, tt.want)
		}
	}
}
