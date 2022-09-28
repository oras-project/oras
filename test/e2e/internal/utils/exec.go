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
	stdout   []match.Matchable
	stderr   []match.Matchable
	stdin    io.Reader
	binary   string
	workDir  string
	exitCode int
}

// Error returns a default execution expecting error.
func Error() *ExecOption {
	return &ExecOption{exitCode: 1}
}

// Success returns a default execution expecting success.
func Success() *ExecOption {
	return &ExecOption{exitCode: 0}
}

// WithInput adds input to execution option.
func (opts *ExecOption) WithInput(r io.Reader) *ExecOption {
	opts.stdin = r
	return opts
}

// WithStdoutKeyWords adds key word matching to stdout.
func (opts *ExecOption) WithStdoutKeyWords(keywords ...string) *ExecOption {
	opts.stdout = append(opts.stdout, match.Keywords(keywords))
	return opts
}

// WithStderrKeyWords adds key word matching to Stdin.
func (opts *ExecOption) WithStderrKeyWords(keywords ...string) *ExecOption {
	opts.stderr = append(opts.stderr, match.Keywords(keywords))
	return opts
}

// WithContent adds full content matching to the exection option.
func (opts *ExecOption) WithContent(content *string) *ExecOption {
	if opts.exitCode == 0 {
		opts.stdout = append(opts.stdout, match.NewContent(content))
	} else {
		opts.stderr = append(opts.stderr, match.NewContent(content))
	}
	return opts
}

// WithStatus adds full content matching to the exection option.
func (opts *ExecOption) WithStatus(keys []match.StateKey, cmd string, verbose bool, successCount int) *ExecOption {
	opts.stdout = append(opts.stdout, match.NewStatus(keys, cmd, verbose, successCount))
	return opts
}

// Exec helps execute `OrasPath args...` with text as description and o as
// matching option.
func Exec(opts *ExecOption, text string, args ...string) {
	ginkgo.It(opts.description(text, args), func() {
		var cmd *exec.Cmd
		if opts.binary == "" {
			opts.binary = OrasPath
		}

		cmd = exec.Command(opts.binary, args...)
		cmd.Stdin = opts.stdin
		var stdout, stderr *match.Output
		if len(opts.stdout) != 0 {
			stdout = match.NewOutput()
		}
		if len(opts.stderr) != 0 {
			stderr = match.NewOutput()
		}
		session, err := gexec.Start(cmd, stdout, stderr)
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(session, "10s").Should(gexec.Exit(opts.exitCode))
		for _, s := range opts.stdout {
			s.Match(stdout.Content)
		}
		for _, s := range opts.stderr {
			s.Match(stderr.Content)
		}
	})
}
