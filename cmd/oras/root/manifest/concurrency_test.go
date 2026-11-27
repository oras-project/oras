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

import (
	"strings"
	"testing"
)

// TestManifestPushRejectsNegativeConcurrency checks the manifest push path,
// which also hands --concurrency to the oras-go tag options.
func TestManifestPushRejectsNegativeConcurrency(t *testing.T) {
	cmd := pushCmd()
	if err := cmd.Flags().Parse([]string{"--concurrency", "-3"}); err != nil {
		t.Fatalf("parsing flags: %v", err)
	}
	err := cmd.PreRunE(cmd, []string{"localhost:5000/hello:v1", "manifest.json"})
	if err == nil {
		t.Fatal("expected an error for a negative concurrency level")
	}
	want := "`--concurrency -3` is invalid: concurrency level must not be negative"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("got %q, want it to contain %q", err, want)
	}
}
