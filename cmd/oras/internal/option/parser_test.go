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
	"errors"
	"testing"

	"oras.land/oras/cmd/oras/internal/option"
)

type Test struct {
	CntPtr *int
}

func (t *Test) Parse() error {
	*t.CntPtr += 1
	if *t.CntPtr == 2 {
		return errors.New("should not be tried twice")
	}
	return nil
}

func TestParse_once(t *testing.T) {
	cnt := 0
	type args struct {
		Test
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"parse should be called once", args{Test{CntPtr: &cnt}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := option.Parse(&tt.args); (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if cnt != 1 {
				t.Errorf("Expect Parse() to be called once but got %v", cnt)
			}
		})
	}
}

func TestParse_err(t *testing.T) {
	cnt := 0
	type args struct {
		Test1 Test
		Test2 Test
		Test3 Test
		Test4 Test
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"parse should be called twice and aborted with error", args{Test{CntPtr: &cnt}, Test{CntPtr: &cnt}, Test{CntPtr: &cnt}, Test{CntPtr: &cnt}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := option.Parse(&tt.args); (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if cnt != 2 {
				t.Errorf("Expect Parse() to be called twice but got %v", cnt)
			}
		})
	}
}
