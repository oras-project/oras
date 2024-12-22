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

package trace

import (
	"reflect"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestTextFormatter_Format(t *testing.T) {
	tests := []struct {
		name    string
		f       *TextFormatter
		entry   *logrus.Entry
		want    []byte
		wantErr bool
	}{
		{
			name: "debug log entry",
			f:    &TextFormatter{},
			entry: &logrus.Entry{
				Time:    time.Date(2024, time.December, 1, 23, 30, 1, 55, time.UTC),
				Level:   logrus.DebugLevel,
				Message: "test debug",
				Data:    logrus.Fields{},
			},
			want:    []byte("[2024-12-01T23:30:01.000000055Z][DEBUG]: test debug\n\n\n"),
			wantErr: false,
		},
		{
			name: "info log entry",
			f:    &TextFormatter{},
			entry: &logrus.Entry{
				Time:    time.Date(2024, time.December, 1, 23, 30, 1, 55, time.UTC),
				Level:   logrus.InfoLevel,
				Message: "test info",
				Data:    logrus.Fields{},
			},
			want:    []byte("[2024-12-01T23:30:01.000000055Z][INFO]: test info\n\n\n"),
			wantErr: false,
		},
		{
			name: "warning log entry with data fields",
			f:    &TextFormatter{},
			entry: &logrus.Entry{
				Time:    time.Date(2024, time.December, 1, 23, 30, 1, 55, time.UTC),
				Level:   logrus.WarnLevel,
				Message: "test warning with fields",
				Data: logrus.Fields{
					"testkey": "testval",
				},
			},
			want:    []byte("[2024-12-01T23:30:01.000000055Z][WARNING]: test warning with fields\n[Data]:\n  testkey=testval\n\n\n"),
			wantErr: false,
		},
		{
			name: "error log entry with data fields",
			f:    &TextFormatter{},
			entry: &logrus.Entry{
				Time:    time.Date(2024, time.December, 1, 23, 30, 1, 55, time.UTC),
				Level:   logrus.ErrorLevel,
				Message: "test warning with fields",
				Data: logrus.Fields{
					"testkey": 123,
				},
			},
			want:    []byte("[2024-12-01T23:30:01.000000055Z][ERROR]: test warning with fields\n[Data]:\n  testkey=123\n\n\n"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.f.Format(tt.entry)
			if (err != nil) != tt.wantErr {
				t.Errorf("TextFormatter.Format() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TextFormatter.Format() = %s, want %s", got, tt.want)
			}
		})
	}
}
