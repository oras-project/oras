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

package utils

import (
	"fmt"
	"strings"

	"github.com/onsi/gomega"
)

// MatchDefaultFlagValue checks if the flag is found in the output and has the
// default value.
func MatchDefaultFlagValue(flag string, defaultValue string, CommandPath ...string) {
	CommandPath = append(CommandPath, "-h")
	out := ORAS(CommandPath...).Exec().Out.Contents()
	_, help, _ := strings.Cut(string(out), "Flags:")
	var found bool
	_, help, found = strings.Cut(help, fmt.Sprintf("--%s", flag))
	gomega.Expect(found).Should(gomega.BeTrue(), "%q not found in %q", flag, help)
	help, _, _ = strings.Cut(help, "\n  -")
	help, _, _ = strings.Cut(help, "\n      --")
	gomega.Expect(help).Should(gomega.ContainSubstring(fmt.Sprintf("(default %q)", defaultValue)))
}
