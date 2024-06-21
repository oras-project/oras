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
	"strings"
	"testing"
)

func Test_PrintPrettyJSON(t *testing.T) {
	builder := &strings.Builder{}
	given := map[string]int{"bob": 5}
	expected := "{\n  \"bob\": 5\n}\n"
	err := PrintPrettyJSON(builder, given)
	if err != nil {
		t.Error("Expected no error got <" + err.Error() + ">")
	}
	actual := builder.String()
	if expected != actual {
		t.Error("Expected <" + expected + "> not equal to actual <" + actual + ">")
	}
}

func Test_PrintJSON(t *testing.T) {
	builder := &strings.Builder{}
	given := []byte("{\"bob\":5}")
	expected := "{\n  \"bob\": 5\n}\n"
	err := PrintJSON(builder, given, true)
	if err != nil {
		t.Error("Expected no error got <" + err.Error() + ">")
	}
	actual := builder.String()
	if expected != actual {
		t.Error("Expected <" + expected + "> not equal to actual <" + actual + ">")
	}
}

func Test_PrintJSON_ugly(t *testing.T) {
	builder := &strings.Builder{}
	given := []byte("{\"bob\":5}")
	expected := "{\"bob\":5}"
	err := PrintJSON(builder, given, false)
	if err != nil {
		t.Error("Expected no error got <" + err.Error() + ">")
	}
	actual := builder.String()
	if expected != actual {
		t.Error("Expected <" + expected + "> not equal to actual <" + actual + ">")
	}
}

func Test_ToMap(t *testing.T) {
	type test struct {
		Name   string `json:"name"`
		Number int    `json:"number"`
	}
	given := test{Name: "bob", Number: 5}
	actual, err := ToMap(given)
	if err != nil {
		t.Error("Expected no error got <" + err.Error() + ">")
	}
	value, ok := actual["name"]
	if ok == false {
		t.Errorf("Expected key name does not exist %v", actual)
	}
	if value != "bob" {
		t.Errorf("Expected value bob not equal to actual %v", value)
	}
	value, ok = actual["number"]
	if ok == false {
		t.Errorf("Expected key number does not exist %v", actual)
	}
	if value != 5.0 {
		t.Errorf("Expected value 5 not equal to actual %v", value)
	}
	for k := range actual {
		switch k {
		case "name":
		case "number":
		default:
			t.Error("Expected key name or number not equal to actual <" + k + ">")
		}
	}
}

func Test_ToMap_error(t *testing.T) {
	type testError struct {
		Name  string     `json:"name"`
		Cycle *testError `json:"cycle"`
	}
	given := testError{Name: "bob"}
	given.Cycle = &given
	_, err := ToMap(given)
	if err == nil {
		t.Error("Expected error")
	}
}
