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
	"os"
	"os/exec"
	"strings"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"oras.land/oras/test/e2e/internal/utils/match"
)

const default_binary = "oras"

// execOption provides option used to execute a command.
type execOption struct {
	stdout     []match.Matchable
	stderr     []match.Matchable
	stdin      io.Reader
	binary     string
	args       []string
	timeout    time.Duration
	workDir    *string
	shouldFail bool
}

// Error returns a default execution expecting error.
func Error(args ...string) *execOption {
	return &execOption{
		binary:     default_binary,
		args:       args,
		timeout:    10 * time.Second,
		shouldFail: true,
	}
}

// Success returns a default execution expecting success.
func Success(args ...string) *execOption {
	return &execOption{
		binary:     default_binary,
		args:       args,
		timeout:    10 * time.Second,
		shouldFail: false,
	}
}

// WithTimeOut sets timeout for execution.
func (opts *execOption) WithTimeOut(timeout time.Duration) *execOption {
	opts.timeout = timeout
	return opts
}

// WithBinary sets binary for execution.
func (opts *execOption) WithBinary(path string) *execOption {
	opts.binary = path
	return opts
}

// WithWorkDir sets working directory for the execution.
func (opts *execOption) WithWorkDir(path *string) *execOption {
	opts.workDir = path
	return opts
}

// WithInput adds input to execution option.
func (opts *execOption) WithInput(r io.Reader) *execOption {
	opts.stdin = r
	return opts
}

// MatchKeyWords adds key word matching to stdout.
func (opts *execOption) MatchKeyWords(keywords ...string) *execOption {
	opts.stdout = append(opts.stdout, match.NewKeywordMatcher(keywords))
	return opts
}

// MatchErrKeyWords adds key word matching to stderr.
func (opts *execOption) MatchErrKeyWords(keywords ...string) *execOption {
	opts.stderr = append(opts.stderr, match.NewKeywordMatcher(keywords))
	return opts
}

// MatchContent adds full content matching to the execution option.
func (opts *execOption) MatchContent(content *string) *execOption {
	if !opts.shouldFail {
		opts.stdout = append(opts.stdout, match.NewContentMatcher(content))
	} else {
		opts.stderr = append(opts.stderr, match.NewContentMatcher(content))
	}
	return opts
}

// MatchStatus adds full content matching to the execution option.
func (opts *execOption) MatchStatus(keys []match.StateKey, verbose bool, successCount int) *execOption {
	opts.stdout = append(opts.stdout, match.NewStatusMatcher(keys, opts.args[0], verbose, successCount))
	return opts
}

// Exec helps execute `OrasPath args...` with text as description and o as
// matching option.
func (opts *execOption) Exec(text string) {
	if opts == nil {
		panic("Nil option for command execution")
	}

	description := fmt.Sprintf("%s: %s %s", text, opts.binary, strings.Join(opts.args, " "))
	ginkgo.By(description, func() {
		var cmd *exec.Cmd
		if opts.binary == default_binary {
			opts.binary = ORASPath
		}

		cmd = exec.Command(opts.binary, opts.args...)
		cmd.Stdin = opts.stdin
		if opts.workDir != nil {
			wd, err := os.Getwd()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(os.Chdir(*opts.workDir)).ShouldNot(HaveOccurred())
			defer os.Chdir(wd)
		}

		session, err := gexec.Start(cmd, nil, nil)
		Expect(err).ShouldNot(HaveOccurred())

		if opts.shouldFail {
			Expect(session.Wait(opts.timeout).ExitCode()).ShouldNot(Equal(0))
		} else {
			Expect(session.Wait(opts.timeout).ExitCode()).Should(Equal(0))
		}

		for _, s := range opts.stdout {
			s.Match(session.Out)
		}
		for _, s := range opts.stderr {
			s.Match(session.Err)
		}
	})
}
