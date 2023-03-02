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
	. "github.com/onsi/ginkgo/v2"
	. "oras.land/oras/test/e2e/internal/utils"
)

var _ = Describe("OCI image layout users:", func() {
	When("custom header is provided", func() {
		It("should fail attach", func() {
			ORAS("attach", ".:test", "-a", "test=true", "--artifact-type", "doc/example", "--oci-layout", "-H=foo:bar").
				WithWorkDir(GinkgoT().TempDir()).
				ExpectFailure().
				MatchErrKeyWords("custom header").
				Exec()
		})
	})
})
