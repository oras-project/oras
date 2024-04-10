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
	"oras.land/oras/test/e2e/internal/testdata/feature"
	"oras.land/oras/test/e2e/internal/testdata/multi_arch"
	. "oras.land/oras/test/e2e/internal/utils"
)

var _ = Describe("ORAS beginners:", func() {
	When("running resolve command", func() {
		It("should show help description with feature mark", func() {
			out := ORAS("resolve", "--help").MatchKeyWords(ExampleDesc).Exec().Out
			gomega.Expect(out.Contents()).Should(gomega.HavePrefix(feature.Experimental.Mark))
		})
		It("should show help description via alias", func() {
			out := ORAS("digest", "--help").MatchKeyWords(ExampleDesc).Exec().Out
			gomega.Expect(out.Contents()).Should(gomega.HavePrefix(feature.Experimental.Mark))
		})
		It("should fail when no manifest reference provided", func() {
			ORAS("resolve").ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})
		It("should fail when repo is invalid", func() {
			ORAS("resolve", fmt.Sprintf("%s/%s", ZOTHost, InvalidRepo)).ExpectFailure().MatchErrKeyWords("Error:", "localhost:7000/INVALID", "invalid reference", "<registry>/<repo>[:tag|@digest]").Exec()
		})
		It("should fail when no tag or digest provided", func() {
			ORAS("resolve", RegistryRef(ZOTHost, ImageRepo, "")).ExpectFailure().MatchErrKeyWords("Error:", `no tag or digest specified`, "oras resolve [flags] <name>{:<tag>|@<digest>}", "Please specify a reference").Exec()
		})
		It("should fail when provided manifest reference is not found", func() {
			ORAS("resolve", RegistryRef(ZOTHost, ImageRepo, "i-dont-think-this-tag-exists")).ExpectFailure().MatchErrKeyWords(RegistryErrorPrefix, "failed to resolve digest:", "not found").Exec()
		})
		It("should fail with empty response when returned response doesn't have body", func() {
			ORAS("resolve", RegistryRef(ZOTHost, ImageRepo, InvalidTag), "-u", Username, "-p", Password+"1").
				ExpectFailure().
				MatchErrKeyWords(RegistryErrorPrefix, EmptyBodyPrefix, "response status code 401: Unauthorized").
				Exec()
		})

		It("should fail and show detailed error description if no argument provided", func() {
			err := ORAS("resolve").ExpectFailure().Exec().Err
			gomega.Expect(err).Should(gbytes.Say("Error"))
			gomega.Expect(err).Should(gbytes.Say("\nUsage: oras resolve"))
			gomega.Expect(err).Should(gbytes.Say("\n"))
			gomega.Expect(err).Should(gbytes.Say(`Run "oras resolve -h"`))
		})
	})
})

var _ = Describe("Common registry user", func() {
	When("running resolve command", func() {
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

var _ = Describe("OCI image layout users", func() {
	When("running resolve command", func() {
		It("should resolve with just digest", func() {
			tmpRoot := PrepareTempOCI(ImageRepo)
			out := ORAS("resolve", Flags.Layout, LayoutRef(tmpRoot, multi_arch.Digest)).Exec().Out
			outString := string(out.Contents())
			outString = strings.TrimSpace(outString)
			gomega.Expect(outString).To(gomega.Equal(multi_arch.Digest))
		})
		It("should resolve with a fully qualified reference", func() {
			tmpRoot := PrepareTempOCI(ImageRepo)
			out := ORAS("digest", Flags.Layout, "-l", LayoutRef(tmpRoot, multi_arch.Tag)).Exec().Out
			gomega.Expect(out).To(gbytes.Say(fmt.Sprintf("%s@%s", tmpRoot, multi_arch.Digest)))
		})
		It("should resolve with a fully qualified reference for a platform", func() {
			tmpRoot := PrepareTempOCI(ImageRepo)
			out := ORAS("resolve", Flags.Layout, "--full-reference", "--platform", "linux/amd64", LayoutRef(tmpRoot, multi_arch.Tag)).Exec().Out
			gomega.Expect(out).To(gbytes.Say(fmt.Sprintf("%s@%s", tmpRoot, multi_arch.LinuxAMD64.Digest)))
		})
	})
})
