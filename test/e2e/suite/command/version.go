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

package command

import (
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"oras.land/oras/test/e2e/internal/utils"
)

var _ = Describe("ORAS user", func() {
	When("checks oras version", func() {
		It("running version command", func() {
			By("should success", func() {
				session, err := gexec.Start(exec.Command(utils.ORASPath, "version"), nil, nil)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(session, "10s").Should(gexec.Exit(0))
			})
		})
	})
})
