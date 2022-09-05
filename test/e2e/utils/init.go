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
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"oras.land/oras-go/v2/registry"
)

var OrasPath string
var Host string
var imageDirPath string
var artifactDirPath string

func init() {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	imageDirPath = filepath.Join(pwd, "..", "testdata", "images")
	artifactDirPath = filepath.Join(pwd, "..", "testdata", "artifacts")
	Host = os.Getenv("ORAS_REGISTRY_HOST")
	if Host == "" {
		Host = "localhost:5000"
		os.Stderr.Write([]byte(fmt.Sprintln("cannot find host name in ORAS_REGISTRY_HOST, using " + Host + " instead")))
	}
	if err := (registry.Reference{Registry: Host}).ValidateRegistry(); Host == "" || err != nil {
		panic(err)
	}
	var _ = BeforeSuite(func() {
		OrasPath = os.Getenv("ORAS_PATH")

		if filepath.IsAbs(OrasPath) {
			// test against OrasPath directly
			fmt.Printf("Testing based on pre-built binary locates in %q\n", OrasPath)
		} else if workspacePath := os.Getenv("GITHUB_WORKSPACE"); filepath.IsAbs(OrasPath) && workspacePath != "" {
			// Add workspacePath as prefix
			OrasPath = filepath.Join(workspacePath, OrasPath)
			OrasPath, err = filepath.Abs(OrasPath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			fmt.Printf("Testing based on pre-built binary locates in %q\n", OrasPath)
		} else {
			// fallback to native build to facilitate locally debugging
			var err error
			OrasPath, err = gexec.Build("oras.land/oras/cmd/oras")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			DeferCleanup(gexec.CleanupBuildArtifacts)
			fmt.Printf("Testing based on temp binary locates in %q\n", OrasPath)
		}
	})

}

func ImageBlob(name string) string {
	return filepath.Join(imageDirPath, name)
}
