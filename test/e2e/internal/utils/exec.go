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
	"io"
	"os/exec"
	"strings"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"oras.land/oras/test/e2e/internal/utils/match"
)

func (opts *ExecOption) description(text string, args []string) string {
	return fmt.Sprintf("%s: %s %s", text, opts.binary, strings.Join(args, " "))
}

// ExecOption provides option used to execute a command.
type ExecOption struct {
	stdout     []match.Matchable
	stderr     []match.Matchable
	stdin      io.Reader
	binary     string
	workDir    string
	shouldFail bool
}

// Exec helps execute `OrasPath args...` with text as description and o as
// matching option.
func Exec(opts ExecOption, text string, args ...string) {
	ginkgo.It(opts.description(text, args), func() {
		var cmd *exec.Cmd
		if opts.binary == "" {
			opts.binary = OrasPath
		}

		cmd = exec.Command(opts.binary, args...)
		cmd.Stdin = opts.stdin
		var stdout, stderr io.Writer
		if len(opts.stdout) != 0 {
			stdout = match.NewOutput()
		}
		if len(opts.stderr) != 0 {
			stderr = match.NewOutput()
		}
		session, err := gexec.Start(cmd, stdout, stderr)
		Expect(err).ShouldNot(HaveOccurred())

		exitCode := 0
		if opts.shouldFail {
			exitCode = 1
		}

		Eventually(session, "10s").Should(gexec.Exit(exitCode))
		for _, s := range opts.stdout {
			s.Match(stdout)
		}
		for _, s := range opts.stderr {
			s.Match(stderr)
		}
	})
}
