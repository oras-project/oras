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

package option_test

import (
	"testing"

	"oras.land/oras/cmd/oras/internal/option"
)

type Test struct {
	Cnt int
}

func (t *Test) Parse() {
	t.Cnt++
}

func TestParse(t *testing.T) {
	type args struct {
		Test
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"parse should be called", args{Test{Cnt: 0}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := option.Parse(&tt.args); (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.args.Cnt != 1 {
				t.Errorf("Expect Parse() to be called once but got %v", tt.args.Cnt)
			}
		})
	}
}
