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
	"os/exec"
	"strings"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"oras.land/oras/test/e2e/utils/match"
)

func description(text string, args []string) string {
	return fmt.Sprintf("%s: oras %s", text, strings.Join(args, " "))
}

// Exec helps execute `OrasPath args...` with text as description and o as
// matching option.
func Exec(o *match.Option, text string, args ...string) {
	ginkgo.It(description(text, args), func() {
		cmd := exec.Command(OrasPath, args...)
		if o.Stdin != nil {
			cmd.Stdin = o.Stdin
		}
		session, err := gexec.Start(cmd, o.Stdout.Writer(), o.Stderr.Writer())
		Expect(err).ShouldNot(HaveOccurred())

		exitCode := 0
		if o.ShouldFail {
			exitCode = 1
		}
		Eventually(session, "10s").Should(gexec.Exit(exitCode))
		o.Match()
	})
}
