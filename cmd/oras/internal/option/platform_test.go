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
	"reflect"
	"runtime"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/pflag"
)

func TestPlatform_ApplyFlags(t *testing.T) {
	var test struct{ Platform }
	ApplyFlags(&test, pflag.NewFlagSet("oras-test", pflag.ExitOnError))
	if test.Platform.platform != "" {
		t.Fatalf("expecting platform to be empty but got: %v", test.Platform.platform)
	}
}

func TestPlatform_Parse_err(t *testing.T) {
	tests := []struct {
		name string
		opts *Platform
	}{
		{name: "empty arch 1", opts: &Platform{"os/", nil}},
		{name: "empty arch 2", opts: &Platform{"os//variant", nil}},
		{name: "empty os", opts: &Platform{"/arch", nil}},
		{name: "empty os with variant", opts: &Platform{"/arch/variant", nil}},
		{name: "trailing slash", opts: &Platform{"os/arch/variant/llama", nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Parse()
			if err == nil {
				t.Errorf("Platform.Parse() error = %v, wantErr %v", err, true)
				return
			}
		})
	}
}

func TestPlatform_Parse(t *testing.T) {
	tests := []struct {
		name string
		opts *Platform
		want *ocispec.Platform
	}{
		{name: "empty", opts: &Platform{platform: ""}, want: nil},
		{name: "default arch", opts: &Platform{platform: "os"}, want: &ocispec.Platform{OS: "os", Architecture: runtime.GOARCH}},
		{name: "os&arch", opts: &Platform{platform: "os/aRcH"}, want: &ocispec.Platform{OS: "os", Architecture: "aRcH"}},
		{name: "empty variant", opts: &Platform{platform: "os/aRcH/"}, want: &ocispec.Platform{OS: "os", Architecture: "aRcH", Variant: ""}},
		{name: "os&arch&variant", opts: &Platform{platform: "os/aRcH/vAriAnt"}, want: &ocispec.Platform{OS: "os", Architecture: "aRcH", Variant: "vAriAnt"}},
		{name: "os version", opts: &Platform{platform: "os/aRcH/vAriAnt:osversion"}, want: &ocispec.Platform{OS: "os", Architecture: "aRcH", Variant: "vAriAnt", OSVersion: "osversion"}},
		{name: "long os version", opts: &Platform{platform: "os/aRcH"}, want: &ocispec.Platform{OS: "os", Architecture: "aRcH"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.opts.Parse(); err != nil {
				t.Errorf("Platform.Parse() error = %v", err)
			}
			got := tt.opts.Platform
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Platform.Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}
