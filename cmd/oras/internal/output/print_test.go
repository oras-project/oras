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

package output

import (
	"fmt"
	"os"
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
	return 0, fmt.Errorf("boom %s", string(p))
}

func (mw *mockWriter) String() string {
	return mw.written
}

func TestPrinter_Println(t *testing.T) {
	mockWriter := &mockWriter{}
	printer := NewPrinter(mockWriter, os.Stderr)
	err := printer.Println("boom")
	if mockWriter.errorCount != 1 {
		t.Error("Expected one error actual <" + strconv.Itoa(mockWriter.errorCount) + ">")
	}
	if err == nil {
		t.Error("Expected error got <nil>")
	}
	err = printer.Printf("boom")
	if mockWriter.errorCount != 2 {
		t.Error("Expected two errors actual <" + strconv.Itoa(mockWriter.errorCount) + ">")
	}
	if err != nil {
		t.Error("Expected error to be ignored")
	}
}

func TestPrinter_PrintVerbose_noError(t *testing.T) {
	builder := &strings.Builder{}
	printer := NewPrinter(builder, os.Stderr)

	expected := "normal\nthing one\n"
	err := printer.Println("normal")
	if err != nil {
		t.Error("Expected no error got <" + err.Error() + ">")
	}
	err = printer.Printf("thing %s\n", "one")
	if err != nil {
		t.Error("Expected no error got <" + err.Error() + ">")
	}
	err = printer.PrintVerbose("verbose")
	if err != nil {
		t.Error("Expected no error got <" + err.Error() + ">")
	}
	actual := builder.String()
	if expected != actual {
		t.Error("Expected <" + expected + "> not equal to actual <" + actual + ">")
	}
}

func TestPrinter_PrintVerbose(t *testing.T) {
	builder := &strings.Builder{}
	printer := NewPrinter(builder, os.Stderr)
	printer.Verbose = true

	expected := "normal\nverbose\n"
	err := printer.Println("normal")
	if err != nil {
		t.Error("Expected no error got <" + err.Error() + ">")
	}
	err = printer.PrintVerbose("verbose")
	if err != nil {
		t.Error("Expected no error got <" + err.Error() + ">")
	}
	actual := builder.String()
	if expected != actual {
		t.Error("Expected <" + expected + "> not equal to actual <" + actual + ">")
	}
}
