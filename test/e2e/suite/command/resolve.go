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
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"oras.land/oras/test/e2e/internal/testdata/multi_arch"
	. "oras.land/oras/test/e2e/internal/utils"
)

var _ = Describe("ORAS beginners:", func() {
	When("running resolve command", func() {
		It("should fail when no manifest reference provided", func() {
			ORAS("resolve").ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})
		It("should fail when repo is invalid", func() {
			ORAS("resolve", fmt.Sprintf("%s/%s", ZOTHost, InvalidRepo)).ExpectFailure().MatchErrKeyWords("Error:", fmt.Sprintf("invalid reference: invalid repository %q", InvalidRepo)).Exec()
		})
		It("should fail when no tag or digest provided", func() {
			ORAS("resolve", RegistryRef(ZOTHost, ImageRepo, "")).ExpectFailure().MatchErrKeyWords("Error:", "no tag or digest when expecting <name:tag|name@digest>").Exec()
		})
		It("should fail when provided manifest reference is not found", func() {
			ORAS("resolve", RegistryRef(ZOTHost, ImageRepo, "i-dont-think-this-tag-exists")).ExpectFailure().MatchErrKeyWords("Error: failed to resolve digest:", "not found").Exec()
		})
		It("should resolve with just digest", func() {
			out := ORAS("resolve", RegistryRef(ZOTHost, ImageRepo, multi_arch.Digest)).Exec().Out
			outString := string(out.Contents())
			outString = strings.TrimSpace(outString)
			gomega.Expect(outString).To(gomega.Equal(multi_arch.Digest))
		})
		It("should resolve with a fully qualified reference", func() {
			out := ORAS("digest", "-l", RegistryRef(ZOTHost, ImageRepo, multi_arch.Tag)).Exec().Out
			gomega.Expect(out).To(gbytes.Say(fmt.Sprintf("%s/%s@%s", ZOTHost, ImageRepo, multi_arch.Digest)))
		})
		It("should resolve with a fully qualified reference for a platform", func() {
			out := ORAS("resolve", "--full-reference", "--platform", "linux/amd64", RegistryRef(ZOTHost, ImageRepo, multi_arch.Tag)).Exec().Out
			gomega.Expect(out).To(gbytes.Say(fmt.Sprintf("%s/%s@%s", ZOTHost, ImageRepo, multi_arch.LinuxAMD64.Digest)))
		})
	})
})
