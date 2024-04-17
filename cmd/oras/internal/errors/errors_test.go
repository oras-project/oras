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

package errors

import (
	"testing"

	"github.com/spf13/pflag"
)

func TestCheckMutuallyExclusiveFlags(t *testing.T) {
	fs := &pflag.FlagSet{}
	var foo, bar, hello bool
	fs.BoolVar(&foo, "foo", false, "foo test")
	fs.BoolVar(&bar, "bar", false, "bar test")
	fs.BoolVar(&hello, "hello", false, "hello test")
	fs.Lookup("foo").Changed = true
	fs.Lookup("bar").Changed = true
	tests := []struct {
		name             string
		exclusiveFlagSet []string
		wantErr          bool
	}{
		{
			"--foo and --bar should not be used at the same time",
			[]string{"foo", "bar"},
			true,
		},
		{
			"--foo and --hello are not used at the same time",
			[]string{"foo", "hello"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckMutuallyExclusiveFlags(fs, tt.exclusiveFlagSet); (err != nil) != tt.wantErr {
				t.Errorf("CheckMutuallyExclusiveFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
