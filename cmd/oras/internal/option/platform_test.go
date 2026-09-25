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
	"fmt"
	"reflect"
	"runtime"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/pflag"
)

func TestPlatform_ApplyFlags(t *testing.T) {
	var opts Platform
	fs := pflag.NewFlagSet("oras-test", pflag.ContinueOnError)
	opts.ApplyFlags(fs)
	if err := fs.Parse([]string{"--platform", "linux/amd64"}); err != nil {
		t.Fatal(err)
	}
	if err := opts.Parse(nil); err != nil {
		t.Fatal(err)
	}
	if got := opts.Platform; !reflect.DeepEqual(got, &ocispec.Platform{OS: "linux", Architecture: "amd64"}) {
		t.Fatalf("Platform = %#v, want linux/amd64", got)
	}
}

func TestPlatform_Parse_err(t *testing.T) {
	tests := []string{"os/", "os//variant", "/arch", "/arch/variant", "os/arch/variant/llama"}
	for _, platform := range tests {
		t.Run(platform, func(t *testing.T) {
			opts := &Platform{platform: platform}
			if err := opts.Parse(nil); err == nil {
				t.Error("Platform.Parse() error = nil, want an error")
			}
		})
	}
}

func TestPlatform_Parse(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want *ocispec.Platform
	}{
		{name: "empty", in: "", want: nil},
		{name: "default arch", in: "os", want: &ocispec.Platform{OS: "os", Architecture: runtime.GOARCH}},
		{name: "os and arch", in: "os/aRcH", want: &ocispec.Platform{OS: "os", Architecture: "aRcH"}},
		{name: "empty variant", in: "os/aRcH/", want: &ocispec.Platform{OS: "os", Architecture: "aRcH"}},
		{name: "variant", in: "os/aRcH/vAriAnt", want: &ocispec.Platform{OS: "os", Architecture: "aRcH", Variant: "vAriAnt"}},
		{name: "os version", in: "os/aRcH/vAriAnt:osversion", want: &ocispec.Platform{OS: "os", Architecture: "aRcH", Variant: "vAriAnt", OSVersion: "osversion"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &Platform{platform: tt.in}
			if err := opts.Parse(nil); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(opts.Platform, tt.want) {
				t.Errorf("Platform.Parse() = %#v, want %#v", opts.Platform, tt.want)
			}
		})
	}
}

func TestPlatforms_Parse(t *testing.T) {
	opts := &Platforms{platforms: []string{"linux/amd64", "linux/arm/v7", "windows/amd64:10", "linux/amd64"}}
	if err := opts.Parse(nil); err != nil {
		t.Fatal(err)
	}
	want := []*ocispec.Platform{
		{OS: "linux", Architecture: "amd64"},
		{OS: "linux", Architecture: "arm", Variant: "v7"},
		{OS: "windows", Architecture: "amd64", OSVersion: "10"},
	}
	if !reflect.DeepEqual(opts.Platforms, want) {
		t.Fatalf("Platforms.Parse() = %#v, want %#v", opts.Platforms, want)
	}
}

func TestPlatforms_ApplyFlags(t *testing.T) {
	var opts Platforms
	fs := pflag.NewFlagSet("oras-test", pflag.ContinueOnError)
	opts.ApplyFlags(fs)
	if err := fs.Parse([]string{"--platform", "linux/amd64,linux/arm64", "--platform", "windows/amd64"}); err != nil {
		t.Fatal(err)
	}
	if err := opts.Parse(nil); err != nil {
		t.Fatal(err)
	}
	if got := len(opts.Platforms); got != 3 {
		t.Fatalf("len(Platforms) = %d, want 3", got)
	}
}

func TestPlatforms_Parse_empty(t *testing.T) {
	for i, platforms := range [][]string{{"linux/amd64", ""}, {"linux/amd64", "", "linux/arm64"}} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			opts := &Platforms{platforms: platforms}
			if err := opts.Parse(nil); err == nil {
				t.Error("Platforms.Parse() error = nil, want an error")
			}
		})
	}
}
