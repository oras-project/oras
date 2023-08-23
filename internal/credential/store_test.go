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

package credential

import (
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
)

func TestNewStoreError(t *testing.T) {
	tmpDir := t.TempDir()
	filename := path.Join(tmpDir, "testfile.txt")
	file, err := os.Create(filename)
	if err != nil {
		t.Errorf("error: cannot create file : %v", err)
	}
	defer file.Close()

	err = os.Chmod(filename, 000)
	if err != nil {
		t.Errorf("error: cannot change file permissions: %v", err)
	}
	credStore, err := NewStore(filename)
	if credStore != nil {
		t.Errorf("Expected NewStore to return nil but actually returned %v ", credStore)
	}
	if err != nil {
		ok := strings.Contains(err.Error(), "failed to open config file")
		reflect.DeepEqual(ok, true)

	} else {
		t.Errorf("Expected err to be not nil")
	}
}
