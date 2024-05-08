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

package status

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

type mockWriter struct {
	errorCount int
	written    string
}

func (mw *mockWriter) Write(p []byte) (n int, err error) {
	mw.written += string(p)
	if strings.TrimSpace(string(p)) != "boom" {
		return len(string(p)), nil
	}
	mw.errorCount++
	return 0, fmt.Errorf("Boom: " + string(p))
}

func (mw *mockWriter) String() string {
	return mw.written
}

func TestPrint_Error(t *testing.T) {
	mockWriter := &mockWriter{}
	printer := NewPrinter(mockWriter)
	printer.Println("boom")
	if mockWriter.errorCount != 1 {
		t.Error("Expected one errors actual <" + strconv.Itoa(mockWriter.errorCount) + ">")
	}
}

func TestPrint_NoError(t *testing.T) {
	mockWriter := &mockWriter{}
	printer := NewPrinter(mockWriter)

	expected := "blah blah"
	printer.Println(expected)
	actual := strings.TrimSpace(mockWriter.String())
	if expected != actual {
		t.Error("Expected <" + expected + "> not equal to actual <" + actual + ">")
	}
	if mockWriter.errorCount != 0 {
		t.Error("Expected no errors actual <" + strconv.Itoa(mockWriter.errorCount) + ">")
	}
}
