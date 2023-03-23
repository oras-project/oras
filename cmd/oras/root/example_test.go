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

package root_test

import (
	"fmt"
	"os"
	"strings"

	"oras.land/oras/cmd/oras/root"
)

// Example gives playable snippets of using oras CLI in command-line scripting
// via Golang.
func Example() {
	// change below array to play with your own cmd args:
	args := []string{"repo", "ls", "mcr.microsoft.com"}
	cmd := root.New()
	cmd.SetArgs(args)
	fmt.Printf("Executing 'oras %s':", strings.Join(args, " "))
	err := cmd.Execute()
	if err != nil {
		fmt.Errorf("Failed to execute : %w", err)
		os.Exit(-1)
	}
	os.Exit(0)
}
