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


package root

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func Test_tagCmd_requiresSourceAndDestination(t *testing.T) {
	rootCmd := &cobra.Command{Use: "oras"}
	rootCmd.AddCommand(tagCmd())
	rootCmd.SetArgs([]string{"tag", "localhost:5000/hello:v1"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected an argument validation error")
	}
	if !strings.Contains(err.Error(), "at least 2 arguments") {
		t.Fatalf("expected an argument count error, got %v", err)
	}
}
