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
	"fmt"
	"reflect"
	"testing"

	credentials "github.com/oras-project/oras-credentials-go"
)

func TestNewStoreError(t *testing.T) {
	// save current function
	oldFunction := CreateNewStore
	// restoring changes
	defer func() { CreateNewStore = oldFunction }()
	CreateNewStore = func(configPath string, opts credentials.StoreOptions) (credentials.Store, error) {
		return nil, fmt.Errorf("New Error")
	}
	var config string = "testconfig"
	credStore, err := NewStore(config)
	if !reflect.DeepEqual(credStore, nil) {
		t.Errorf("Expected NewStore to return nil but actually returned %v ", credStore)
	}
	if reflect.DeepEqual(err, nil) {
		t.Error("Expected Error to be not nil but actually returned nil ")
	}
}
