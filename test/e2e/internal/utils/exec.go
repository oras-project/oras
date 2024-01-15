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
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"oras.land/oras/test/e2e/internal/utils/match"
)

const (
	orasBinary = "oras"

	// customize your own basic auth file via `htpasswd -cBb <file_name> <user_name> <password>`
	Username         = "hello"
	Password         = "oras-test"
	DefaultTimeout   = 10 * time.Second
	// If the command hasn't exited yet, ginkgo session ExitCode is -1
	notResponding = -1
)

// ExecOption provides option used to execute a command.
type ExecOption struct {
	binary  string
	args    []string
	workDir string
	timeout time.Duration

	stdin    io.Reader
	stdout   []match.Matcher
	stderr   []match.Matcher
	exitCode int

	text string
}

// ORAS returns default execution option for oras binary.
func ORAS(args ...string) *ExecOption {
	return Binary(orasBinary, args...)
}

// Binary returns default execution option for customized binary.
func Binary(path string, args ...string) *ExecOption {
	return &ExecOption{
		binary:   path,
		args:     args,
		timeout:  DefaultTimeout,
		exitCode: 0,
	}
}

// ExpectFailure sets failure exit code checking for the execution.
func (opts *ExecOption) ExpectFailure() *ExecOption {
	// set to 1 but only check if it's positive
	opts.exitCode = 1
	return opts
}

// ExpectBlocking consistently check if the execution is blocked.
func (opts *ExecOption) ExpectBlocking() *ExecOption {
	opts.exitCode = notResponding
	return opts
}

// WithTimeOut sets timeout for the execution.
func (opts *ExecOption) WithTimeOut(timeout time.Duration) *ExecOption {
	opts.timeout = timeout
	return opts
}

// WithDescription sets description text for the execution.
func (opts *ExecOption) WithDescription(text string) *ExecOption {
	opts.text = text
	return opts
}

// WithWorkDir sets working directory for the execution.
func (opts *ExecOption) WithWorkDir(path string) *ExecOption {
	opts.workDir = path
	return opts
}

// WithInput redirects stdin to r for the execution.
func (opts *ExecOption) WithInput(r io.Reader) *ExecOption {
	opts.stdin = r
	return opts
}

// MatchKeyWords adds keywords matching to stdout.
func (opts *ExecOption) MatchKeyWords(keywords ...string) *ExecOption {
	opts.stdout = append(opts.stdout, match.NewKeywordMatcher(keywords))
	return opts
}

// MatchErrKeyWords adds keywords matching to stderr.
func (opts *ExecOption) MatchErrKeyWords(keywords ...string) *ExecOption {
	opts.stderr = append(opts.stderr, match.NewKeywordMatcher(keywords))
	return opts
}

// MatchRequestHeaders adds a debug log matcher that checks for
// the presence of provided headers in all outgoing HTTP requests.
func (opts *ExecOption) MatchRequestHeaders(headers ...string) *ExecOption {
	opts.stderr = append(opts.stderr, match.NewRequestHeaderMatcher("", headers))
	return opts
}

// MatchCpRequestHeaders adds a debug log matcher that checks for
// the presence of specific headers in outgoing copy requests towards
// the provided host and repository.
func (opts *ExecOption) MatchCpRequestHeaders(host string, repo string, headers ...string) *ExecOption {
	urlPrefix := fmt.Sprintf("://%s/v2/%s/", host, repo)
	opts.stderr = append(opts.stderr, match.NewRequestHeaderMatcher(urlPrefix, headers))
	return opts
}

// MatchContent adds full content matching to the execution.
func (opts *ExecOption) MatchContent(content string) *ExecOption {
	if opts.exitCode == 0 {
		opts.stdout = append(opts.stdout, match.NewContentMatcher(content, false))
	} else {
		opts.stderr = append(opts.stderr, match.NewContentMatcher(content, false))
	}
	return opts
}

// MatchTrimedContent adds trimmed content matching to the execution.
func (opts *ExecOption) MatchTrimmedContent(content string) *ExecOption {
	if opts.exitCode == 0 {
		opts.stdout = append(opts.stdout, match.NewContentMatcher(content, true))
	} else {
		opts.stderr = append(opts.stderr, match.NewContentMatcher(content, true))
	}
	return opts
}

// MatchStatus adds full content matching to the execution option.
func (opts *ExecOption) MatchStatus(keys []match.StateKey, verbose bool, successCount int) *ExecOption {
	opts.stdout = append(opts.stdout, match.NewStatusMatcher(keys, opts.args[0], verbose, successCount))
	return opts
}

// Exec run the execution based on opts.
func (opts *ExecOption) Exec() *gexec.Session {
	if opts == nil {
		// this should be a code error but can only be caught during runtime
		panic("Nil option for command execution")
	}

	if opts.text == "" {
		// set default description text
		switch opts.exitCode {
		case notResponding:
			opts.text = "block"
		case 0:
			opts.text = "pass"
		default:
			opts.text = "fail"
		}
	}
	description := fmt.Sprintf("\n>> should %s: %s %s >>", opts.text, opts.binary, strings.Join(opts.args, " "))
	ginkgo.By(description)

	var cmd *exec.Cmd
	if opts.binary == orasBinary {
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
	if opts.exitCode == notResponding {
		Consistently(session.ExitCode).WithTimeout(opts.timeout).Should(Equal(notResponding))
		session.Kill()
	} else {
		exitCode := session.Wait(opts.timeout).ExitCode()
		Expect(opts.exitCode == 0).To(Equal(exitCode == 0))
	}

	// matching result
	stdout := session.Out.Contents()
	for _, s := range opts.stdout {
		s.Match(gbytes.BufferWithBytes(stdout))
	}

	stderr := session.Err.Contents()
	for _, s := range opts.stderr {
		s.Match(gbytes.BufferWithBytes(stderr))
	}

	return session
}
