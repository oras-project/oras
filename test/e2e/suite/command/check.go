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
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"oras.land/oras/test/e2e/internal/testdata/graph"
	. "oras.land/oras/test/e2e/internal/utils"
)

var _ = Describe("ORAS beginners:", func() {
	When("running `check`", func() {
		It("should show examples in help doc", func() {
			out := ORAS("check", "--help").MatchKeyWords(ExampleDesc).Exec().Out
			gomega.Expect(out).ShouldNot(gbytes.Say("--verbose"))
		})
	})
})

var _ = Describe("1.1 registry users:", func() {
	When("running `check`", func() {
		It("should check a valid image with no errors", func() {
			goodImageRef := RegistryRef(ZOTHost, GraphRepo, graph.GoodImageTag)
			ORAS("check", goodImageRef).MatchKeyWords("Checked", goodImageRef).Exec()
		})
		It("should check the subject field if subject is present", func() {
			goodImageReferrerRef := RegistryRef(ZOTHost, GraphRepo, graph.GoodImageReferrerDigest)
			ORAS("check", goodImageReferrerRef).MatchKeyWords("Checked", graph.GoodImageShortDigest, goodImageReferrerRef).Exec()
		})

	})
})

var _ = Describe("OCI image layout users:", func() {
	When("running `check`", func() {
		It("should check a valid image with no errors", func() {
			// prepare
			goodImageRef := LayoutRef(PrepareTempOCI(GraphRepo), graph.GoodImageTag)
			// test
			ORAS("check", Flags.Layout, goodImageRef).MatchKeyWords("Checked", goodImageRef).Exec()
		})
		It("should check the subject field if subject is present", func() {
			// prepare
			goodImageReferrerRef := LayoutRef(PrepareTempOCI(GraphRepo), graph.GoodImageReferrerDigest)
			// test
			ORAS("check", Flags.Layout, goodImageReferrerRef).MatchKeyWords("Checked", graph.GoodImageShortDigest, goodImageReferrerRef).Exec()
		})
	})
})
