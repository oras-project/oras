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

package matcher

import (
	"fmt"
	"io"
	"os"
	"strings"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

func pipe(ptrFrom **os.File) (io.WriteCloser, io.Reader, func()) {
	saved := *ptrFrom
	var err error
	to, new, err := os.Pipe()
	Expect(err).To(BeNil(), "Should succeed to prepare std our streams")
	*ptrFrom = new
	return new, to, func() {
		*ptrFrom = saved
		defer new.Close()
		defer to.Close()
	}
}

func MatchOut(text string, cmd *cobra.Command, output string) {
	var r io.Reader
	var wc io.WriteCloser
	var close func()

	ginkgo.It(text, func() {
		wc, r, close = pipe(&os.Stdout)
		defer close()

		Expect(cmd.Execute()).Should(Succeed())
		wc.Close()
		b, err := io.ReadAll(r)
		Expect(err).To(BeNil(), "Failed to read command execution output")
		str := string(b)
		Expect(str).To(Equal(output))
	})
}

func MatchOutWithKeyWords(text string, cmd *cobra.Command, keywords []string) {
	var r io.Reader
	var wc io.WriteCloser
	var close func()

	ginkgo.It(text, func() {
		wc, r, close = pipe(&os.Stdout)
		defer close()

		visited := make(map[string]bool)
		for _, w := range keywords {
			visited[w] = false
		}
		Expect(cmd.Execute()).Should(Succeed())
		wc.Close()
		b, err := io.ReadAll(r)
		Expect(err).To(BeNil(), "Failed to read command execution output")

		str := string(b)
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

func MatchErrWithKeyWords(text string, cmd *cobra.Command, keywords []string) {
	var r io.Reader
	var wc io.WriteCloser
	var close func()

	ginkgo.It(text, func() {
		wc, r, close = pipe(&os.Stderr)
		defer close()

		visited := make(map[string]bool)
		for _, w := range keywords {
			visited[w] = false
		}
		Expect(cmd.Execute()).ShouldNot(Succeed())
		wc.Close()
		b, err := io.ReadAll(r)
		Expect(err).To(BeNil(), "Failed to read command error output")

		str := string(b)
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
