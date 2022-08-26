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

package test

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func ExecAndMatchOut(text string, cmdName *string, args []string, output string) {
	stdout := NewWriter()

	ginkgo.It(text, func() {
		session, err := gexec.Start(exec.Command(*cmdName, args...), stdout, io.Discard)
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(session, "30s").Should(gexec.Exit(0))
		Expect(stdout.ReadAll()).To(Equal([]byte(output)))
	})
}

func ExecAndMatchOutKeyWords(text string, cmdName *string, args []string, keywords []string) {
	stdout := NewWriter()

	ginkgo.It(text, func() {
		session, err := gexec.Start(exec.Command(*cmdName, args...), stdout, io.Discard)
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(session).Should(gexec.Exit(0))

		visited := make(map[string]bool)
		for _, w := range keywords {
			visited[w] = false
		}
		str := string(stdout.ReadAll())
		for k := range visited {
			if strings.Contains(str, k) {
				delete(visited, k)
			}
		}

		if len(visited) != 0 {
			Expect(fmt.Sprintf("%v", visited)).To(Equal(""))
		}
	})
}

func ExecAndMatchErrKeyWords(text string, cmdName *string, args []string, keywords []string) {
	stderr := NewWriter()

	ginkgo.It(text, func() {
		session, err := gexec.Start(exec.Command(*cmdName, args...), io.Discard, stderr)
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(session).Should(gexec.Exit(1))

		visited := make(map[string]bool)
		for _, w := range keywords {
			visited[w] = false
		}
		str := string(stderr.ReadAll())
		for k := range visited {
			if strings.Contains(str, k) {
				delete(visited, k)
			}
		}

		if len(visited) != 0 {
			Expect(fmt.Sprintf("%v", visited)).To(Equal(""))
		}
	})
}
