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

package version

import (
	"reflect"
	"testing"
)

func Test_GetVersion_when_BuildData_is_empty(t *testing.T) {
	BuildMetadata = ""
	expected := Version
	actual := GetVersion()
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Tested GetVersion expected was %v actually got %v", expected, actual)
	}
}
