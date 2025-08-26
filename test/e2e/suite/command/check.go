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

const (
	referrerDigest = "sha256:f85dd10854364a7e4940bb65c6f2cce72c6b563618bc81f3ac08202a54ea22ee"
)

var _ = Describe("ORAS beginners:", func() {
	When("running `check`", func() {
		It("should show examples in help doc", func() {
			out := ORAS("check", "--help").MatchKeyWords(ExampleDesc).Exec().Out
			gomega.Expect(out).ShouldNot(gbytes.Say("--verbose"))
		})

		// It("should have flag for including referrers", func() {
		// 	ORAS("blob", "get", "--help").
		// 		MatchKeyWords("--pretty", "prettify JSON").
		// 		Exec()
		// })
	})
})

var _ = Describe("1.1 registry users:", func() {
	//repoFmt := fmt.Sprintf("command/check/%%s/%d/%%s", GinkgoRandomSeed())
	When("running `check`", func() {
		It("should check a valid image with no errors", func() {
			goodImageRef := RegistryRef(ZOTHost, GraphRepo, graph.GoodImageTag)
			ORAS("check", goodImageRef).MatchKeyWords("Checked", goodImageRef).Exec()
		})

		// It("should delete a blob with force flag and output descriptor", func() {
		// 	dstRepo := fmt.Sprintf(repoFmt, "delete", "flag-confirmation")
		// 	CopyZOTRepo(BlobRepo, dstRepo)
		// 	toDeleteRef := RegistryRef(ZOTHost, dstRepo, foobar.FooBlobDigest)
		// 	ORAS("blob", "delete", toDeleteRef, "--force", "--descriptor").MatchContent(foobar.FooBlobDescriptor).Exec()
		// })

	})
})

// var _ = Describe("OCI image layout users:", func() {
// 	When("running `blob delete`", func() {
// 		It("should delete a blob with interactive confirmation", func() {
// 			// prepare
// 			toDeleteRef := LayoutRef(PrepareTempOCI(ImageRepo), foobar.FooBlobDigest)
// 			// test
// 			ORAS("blob", "delete", Flags.Layout, toDeleteRef).
// 				WithInput(strings.NewReader("y")).
// 				MatchKeyWords("Deleted", toDeleteRef).Exec()
// 			// validate
// 			ORAS("blob", "fetch", toDeleteRef, Flags.Layout, "--output", "-").ExpectFailure().Exec()
// 		})
// 	})
// })
