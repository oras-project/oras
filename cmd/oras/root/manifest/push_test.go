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

package manifest

import "testing"

func Test_pushManifestCmd_invalidExtraTag(t *testing.T) {
	cmd := pushCmd()
	cmd.SetArgs([]string{
		"localhost:5000/hello:v1,,invalid@@tag",
		"manifest.json",
	})

	err := cmd.Execute()
	if err == nil || err.Error() != "empty extra tag" {
		t.Fatalf("got %v, want empty extra tag error", err)
	}
}

func Test_pushManifestCmd_invalidHeader(t *testing.T) {
	cmd := pushCmd()
	cmd.SetArgs([]string{
		"localhost:5000/hello:v1",
		"manifest.json",
		"--header", "invalid",
	})

	err := cmd.Execute()
	if err == nil || err.Error() != `invalid header: "invalid"` {
		t.Fatalf("got %v, want invalid header error", err)
	}
}
