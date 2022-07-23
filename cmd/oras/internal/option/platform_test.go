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

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestPlatform_parse_invalid(t *testing.T) {
	var checker = func(flag string) {
		if _, err := (&Platform{flag}).parse(); err == nil {
			t.Fatalf("expecting parse error for flag: %q", flag)
		}
	}

	checker("")
	checker("os/")
	checker("os")
	checker("/arch")
	checker("/arch/variant")
	checker("os/arch/variant/llama")
}

func TestPlatform_parse(t *testing.T) {
	var checker = func(flag string, want ocispec.Platform) {
		got, err := (&Platform{flag}).parse()
		if err != nil {
			t.Fatalf("unexpected parse error for flag: %q", flag)
		}
		if got.OS != want.OS || got.Architecture != want.Architecture || got.Variant != want.Variant || got.OSVersion != want.OSVersion {
			t.Fatalf("Parse result unmatched: expecting %v, got %v", want, got)
		}
	}

	checker("os/aRcH", ocispec.Platform{OS: "os", Architecture: "aRcH"})
	checker("os/aRcH/", ocispec.Platform{OS: "os", Architecture: "aRcH", Variant: ""})
	checker("os/aRcH/vAriAnt", ocispec.Platform{OS: "os", Architecture: "aRcH", Variant: "vAriAnt"})
	checker("os/aRcH/vAriAnt:osversion", ocispec.Platform{OS: "os", Architecture: "aRcH", Variant: "vAriAnt", OSVersion: "osversion"})
}
