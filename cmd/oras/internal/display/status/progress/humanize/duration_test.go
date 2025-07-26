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

package humanize

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{
			name: "zero duration",
			d:    0,
			want: "0s",
		},
		{
			name: "round to microsecond",
			d:    12345 * time.Nanosecond,
			want: "12µs",
		},
		{
			name: "round to millisecond",
			d:    12345 * time.Microsecond,
			want: "12ms",
		},
		{
			name: "round to second",
			d:    12345 * time.Millisecond,
			want: "12s",
		},
		{
			name: "round up to second",
			d:    time.Second + 500*time.Millisecond,
			want: "2s",
		},
		{
			name: "round down to second",
			d:    time.Second + 499*time.Millisecond,
			want: "1s",
		},
		{
			name: "round up to millisecond",
			d:    time.Millisecond + 500*time.Microsecond,
			want: "2ms",
		},
		{
			name: "round down to millisecond",
			d:    time.Millisecond + 499*time.Microsecond,
			want: "1ms",
		},
		{
			name: "round up to microsecond",
			d:    time.Microsecond + 500*time.Nanosecond,
			want: "2µs",
		},
		{
			name: "round down to microsecond",
			d:    time.Microsecond + 499*time.Nanosecond,
			want: "1µs",
		},
		{
			name: "exact second",
			d:    3 * time.Second,
			want: "3s",
		},
		{
			name: "exact millisecond",
			d:    5 * time.Millisecond,
			want: "5ms",
		},
		{
			name: "exact microsecond",
			d:    7 * time.Microsecond,
			want: "7µs",
		},
		{
			name: "less than a microsecond",
			d:    100 * time.Nanosecond,
			want: "0s",
		},
		{
			name: "large duration",
			d:    2*time.Hour + 3*time.Minute + 4*time.Second + 501*time.Millisecond,
			want: "2h3m5s",
		},
		{
			name: "negative duration",
			d:    -5 * time.Second,
			want: "-5s",
		},
		{
			name: "negative small duration",
			d:    -500 * time.Microsecond,
			want: "-500µs",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatDuration(tt.d); got != tt.want {
				t.Errorf("FormatDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}
