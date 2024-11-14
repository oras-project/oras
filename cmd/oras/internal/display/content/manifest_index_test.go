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

package content

import (
	"os"
	"testing"
)

func Test_manifestIndexCreate_OnContentCreated(t *testing.T) {
	testHandler := NewManifestIndexCreateHandler(os.Stdout, false, "invalid/path")
	if err := testHandler.OnContentCreated([]byte("test content")); err == nil {
		t.Errorf("manifestIndexCreate.OnContentCreated() error = %v, wantErr non-nil error", err)
	}
}
