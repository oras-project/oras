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

package progress

import "testing"

func TestRoundTo(t *testing.T) {
	type args struct {
		quantity float64
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{"round to 2 digit", args{1.223}, 1.22},
		{"round to 1 digit", args{12.23}, 12.2},
		{"round to no digit", args{122.6}, 123},
		{"round to no digit", args{1223.123}, 1223},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RoundTo(tt.args.quantity); got != tt.want {
				t.Errorf("RoundTo() = %v, want %v", got, tt.want)
			}
		})
	}
}
