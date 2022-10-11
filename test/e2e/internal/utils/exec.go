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

const oras_binary = "oras"

// execOption provides option used to execute a command.
type execOption struct {
	binary  string
	args    []string
	workDir string
	timeout time.Duration

	stdin      io.Reader
	stdout     []match.Matchable
	stderr     []match.Matchable
	shouldFail bool

	text string
}

// ORAS returns default execution option for oras binary.
func ORAS(args ...string) *execOption {
	return Binary(oras_binary, args...)
}

// Binary returns default execution option for customized binary.
func Binary(path string, args ...string) *execOption {
	return &execOption{
		binary:     path,
		args:       args,
		timeout:    10 * time.Second,
		shouldFail: false,
	}
}

// WithFailureCheck sets failure exit code checking for the execution.
func (opts *execOption) WithFailureCheck() *execOption {
	opts.shouldFail = true
	return opts
}

// WithTimeOut sets timeout for the execution.
func (opts *execOption) WithTimeOut(timeout time.Duration) *execOption {
	opts.timeout = timeout
	return opts
}

// WithDescription sets description text for the execution.
func (opts *execOption) WithDescription(text string) *execOption {
	opts.text = text
	return opts
}

// WithWorkDir sets working directory for the execution.
func (opts *execOption) WithWorkDir(path string) *execOption {
	opts.workDir = path
	return opts
}

// WithInput redirects stdin to r for the execution.
func (opts *execOption) WithInput(r io.Reader) *execOption {
	opts.stdin = r
	return opts
}

// MatchKeyWords adds keywords matching to stdout.
func (opts *execOption) MatchKeyWords(keywords ...string) *execOption {
	opts.stdout = append(opts.stdout, match.NewKeywordMatcher(keywords))
	return opts
}

// MatchErrKeyWords adds keywords matching to stderr.
func (opts *execOption) MatchErrKeyWords(keywords ...string) *execOption {
	opts.stderr = append(opts.stderr, match.NewKeywordMatcher(keywords))
	return opts
}

// MatchContent adds full content matching to the execution.
func (opts *execOption) MatchContent(content string) *execOption {
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

// Exec run the execution based on opts.
func (opts *execOption) Exec() *gexec.Session {
	if opts == nil {
		// this should be a code error but can only be caught during runtime
		panic("Nil option for command execution")
	}

	if opts.text == "" {
		if opts.shouldFail {
			opts.text = "fail"
		} else {
			opts.text = "succeed"
		}
	}
	description := fmt.Sprintf(">> should %s: %s %s >>", opts.text, opts.binary, strings.Join(opts.args, " "))
	ginkgo.By(description)

	var cmd *exec.Cmd
	if opts.binary == oras_binary {
		opts.binary = ORASPath
	}
	cmd = exec.Command(opts.binary, opts.args...)
	cmd.Stdin = opts.stdin
	if opts.workDir != "" {
		// switch working directory
		wd, err := os.Getwd()
		Expect(err).ShouldNot(HaveOccurred())
		Expect(os.Chdir(opts.workDir)).ShouldNot(HaveOccurred())
		defer os.Chdir(wd)
	}
	fmt.Println(description)
	session, err := gexec.Start(cmd, os.Stdout, os.Stderr)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(session.Wait(opts.timeout).ExitCode() != 0).Should(Equal(opts.shouldFail))

	// matching result
	for _, s := range opts.stdout {
		s.Match(session.Out)
	}
	for _, s := range opts.stderr {
		s.Match(session.Err)
	}

	return session
}
