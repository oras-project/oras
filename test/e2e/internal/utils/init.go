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
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"oras.land/oras-go/v2/registry"
)

// ORASPath points to the to-be-tested oras binary.
var ORASPath string

// Host points to the registry service where E2E tests will be run against.
var Host string
var imageDir string

func init() {
	Host = os.Getenv("ORAS_REGISTRY_HOST")
	if Host == "" {
		Host = "localhost:5000"
		fmt.Fprintln(os.Stderr, "cannot find host name in ORAS_REGISTRY_HOST, using", Host, "instead")
	}
	ref := registry.Reference{
		Registry: Host,
	}
	if err := ref.ValidateRegistry(); err != nil {
		panic(err)
	}
	// setup test data
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	imageDir = filepath.Join(pwd, "..", "..", "testdata", "images")
	BeforeSuite(func() {
		ORASPath = os.Getenv("ORAS_PATH")
		if filepath.IsAbs(ORASPath) {
			fmt.Printf("Testing based on pre-built binary locates in %q\n", ORASPath)
			return
		}

		var err error
		if workspacePath := os.Getenv("GITHUB_WORKSPACE"); ORASPath != "" && workspacePath != "" {
			// add workspacePath as prefix, both path env should not be empty
			ORASPath = filepath.Join(workspacePath, ORASPath)
			ORASPath, err = filepath.Abs(ORASPath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			fmt.Printf("Testing based on pre-built binary locates in %q\n", ORASPath)
			return
		}

		// fallback to native build to facilitate local debugging
		ORASPath, err = gexec.Build("oras.land/oras/cmd/oras")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		DeferCleanup(gexec.CleanupBuildArtifacts)
		fmt.Printf("Testing based on temp binary locates in %q\n", ORASPath)
	})
}

var authDone atomic.Bool
var authTry sync.Mutex

// Auth does exactly one `oras login` and blocks return until it finishes or
// fails.
func Auth() {
	JustBeforeEach(func() {
		if authDone.Load() {
			return
		}
		authTry.Lock()
		defer authTry.Unlock()
		if authDone.Load() {
			return
		}
		cmd := exec.Command(ORASPath, "login", Host, "-u", USERNAME, "-p", PASSWORD)
		gomega.Expect(cmd.Run()).ShouldNot(gomega.HaveOccurred())
		authDone.Store(true)
	})
}
