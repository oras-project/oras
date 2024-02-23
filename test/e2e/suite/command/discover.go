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
	"encoding/json"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"gopkg.in/yaml.v2"
	"oras.land/oras/test/e2e/internal/testdata/feature"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	"oras.land/oras/test/e2e/internal/testdata/multi_arch"
	. "oras.land/oras/test/e2e/internal/utils"
)

func discoverKeyWords(verbose bool, descs ...ocispec.Descriptor) []string {
	var ret []string
	for _, d := range descs {
		ret = append(ret, d.Digest.String(), d.ArtifactType)
		if verbose {
			for k, v := range d.Annotations {
				bytes, err := yaml.Marshal(map[string]string{k: v})
				Expect(err).ShouldNot(HaveOccurred())
				ret = append(ret, strings.TrimSpace(string(bytes)))
			}
		}
	}
	return ret
}

var _ = Describe("ORAS beginners:", func() {
	When("running discover command", func() {
		RunAndShowPreviewInHelp([]string{"discover"})

		It("should show preview and help doc", func() {
			out := ORAS("discover", "--help").MatchKeyWords(feature.Preview.Mark+" Discover", feature.Preview.Description, ExampleDesc).Exec()
			gomega.Expect(out).Should(gbytes.Say("--distribution-spec string\\s+%s", regexp.QuoteMeta(feature.Preview.Mark)))
		})

		It("should fail when no subject reference provided", func() {
			ORAS("discover").ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})

		It("should fail when no tag or digest found in provided subject reference", func() {
			ORAS("discover", RegistryRef(ZOTHost, ImageRepo, "")).ExpectFailure().MatchErrKeyWords("Error:", "no tag or digest specified", "oras discover").Exec()
		})

		It("should fail and show detailed error description if no argument provided", func() {
			err := ORAS("discover").ExpectFailure().Exec().Err
			Expect(err).Should(gbytes.Say("Error"))
			Expect(err).Should(gbytes.Say("\nUsage: oras discover"))
			Expect(err).Should(gbytes.Say("\n"))
			Expect(err).Should(gbytes.Say(`Run "oras discover -h"`))
		})

		It("should fail and show detailed error description if more than 1 argument are provided", func() {
			err := ORAS("discover", "foo", "bar").ExpectFailure().Exec().Err
			Expect(err).Should(gbytes.Say("Error"))
			Expect(err).Should(gbytes.Say("\nUsage: oras discover"))
			Expect(err).Should(gbytes.Say("\n"))
			Expect(err).Should(gbytes.Say(`Run "oras discover -h"`))
		})
	})
})

var _ = Describe("1.1 registry users:", func() {
	subjectRef := RegistryRef(ZOTHost, ArtifactRepo, foobar.Tag)
	When("running discover command with json output", func() {
		format := "json"
		It("should discover direct referrers of a subject", func() {
			bytes := ORAS("discover", subjectRef, "-o", format).Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(index.Manifests).To(HaveLen(1))
			Expect(index.Manifests).Should(ContainElement(foobar.SBOMImageReferrer))
		})

		It("should discover matched referrer when filtering", func() {
			bytes := ORAS("discover", subjectRef, "-o", format, "--artifact-type", foobar.SBOMImageReferrer.ArtifactType).Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(index.Manifests).To(HaveLen(1))
			Expect(index.Manifests).Should(ContainElement(foobar.SBOMImageReferrer))
		})

		It("should discover matched no referrer", func() {
			bytes := ORAS("discover", subjectRef, "-o", format, "--artifact-type", "???").Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(index.Manifests).To(HaveLen(0))
		})

		It("should discover one referrer with matched platform", func() {
			bytes := ORAS("discover", RegistryRef(ZOTHost, ArtifactRepo, multi_arch.Tag), "-o", format, "--platform", "linux/amd64").Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(index.Manifests).To(HaveLen(1))
			Expect(index.Manifests).Should(ContainElement(multi_arch.LinuxAMD64Referrer))
		})
	})

	When("running discover command with tree output", func() {
		format := "tree"
		referrers := []ocispec.Descriptor{foobar.SBOMImageReferrer, foobar.SBOMImageReferrer, foobar.SignatureImageReferrer, foobar.SignatureImageReferrer}
		It("should discover all referrers of a subject", func() {
			ORAS("discover", subjectRef, "-o", format).
				MatchKeyWords(append(discoverKeyWords(false, referrers...), RegistryRef(ZOTHost, ArtifactRepo, foobar.Digest))...).
				Exec()
		})

		It("should discover all referrers of a subject via referrers API", func() {
			ORAS("discover", subjectRef, "-o", format, "--distribution-spec", "v1.1-referrers-api").
				MatchKeyWords(append(discoverKeyWords(false, referrers...), RegistryRef(ZOTHost, ArtifactRepo, foobar.Digest))...).
				Exec()
		})

		It("should discover all referrers of a subject with annotations", func() {
			ORAS("discover", subjectRef, "-o", format, "-v").
				MatchKeyWords(append(discoverKeyWords(true, referrers...), RegistryRef(ZOTHost, ArtifactRepo, foobar.Digest))...).
				Exec()

		})
	})
	When("running discover command with table output", func() {
		format := "table"
		It("should all referrers of a subject", func() {
			referrers := []ocispec.Descriptor{foobar.SBOMImageReferrer, foobar.SBOMImageReferrer}
			ORAS("discover", subjectRef, "-o", format).
				MatchKeyWords(append(discoverKeyWords(false, referrers...), foobar.Digest)...).
				Exec()
		})
	})
})

var _ = Describe("1.0 registry users:", func() {
	subjectRef := RegistryRef(FallbackHost, ArtifactRepo, foobar.Tag)
	When("running discover command", func() {
		It("should discover direct referrers of a subject via json output", func() {
			bytes := ORAS("discover", subjectRef, "-o", "json").Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(index.Manifests).To(HaveLen(1))
			Expect(index.Manifests).Should(ContainElement(foobar.SBOMImageReferrer))
		})

		It("should discover matched referrer when filtering via json output", func() {
			bytes := ORAS("discover", subjectRef, "-o", "json", "--artifact-type", foobar.SBOMImageReferrer.ArtifactType).Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(index.Manifests).To(HaveLen(1))
			Expect(index.Manifests).Should(ContainElement(foobar.SBOMImageReferrer))
		})

		It("should discover no referrer when not matching via json output", func() {
			bytes := ORAS("discover", subjectRef, "-o", "json", "--artifact-type", "???").Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(index.Manifests).To(HaveLen(0))
		})

		It("should discover all referrers of a subject via tree output", func() {
			referrers := []ocispec.Descriptor{foobar.SBOMImageReferrer, foobar.SignatureImageReferrer}
			ORAS("discover", subjectRef, "-o", "tree").
				MatchKeyWords(append(discoverKeyWords(false, referrers...), RegistryRef(FallbackHost, ArtifactRepo, foobar.Digest))...).
				Exec()
		})

		It("should discover all referrers with annotation via tree output", func() {
			referrers := []ocispec.Descriptor{foobar.SBOMImageReferrer, foobar.SignatureImageReferrer}
			ORAS("discover", subjectRef, "-o", "tree", "-v").
				MatchKeyWords(append(discoverKeyWords(true, referrers...), RegistryRef(FallbackHost, ArtifactRepo, foobar.Digest))...).
				Exec()
		})

		It("should discover direct referrers of a subject via table output", func() {
			referrers := []ocispec.Descriptor{foobar.SBOMImageReferrer}
			ORAS("discover", subjectRef, "-o", "table").
				MatchKeyWords(append(discoverKeyWords(false, referrers...), foobar.Digest)...).
				Exec()
		})

		It("should discover direct referrers explicitly via tag scheme", func() {
			referrers := []ocispec.Descriptor{foobar.SBOMImageReferrer}
			ORAS("discover", subjectRef, "-o", "table", "--distribution-spec", "v1.1-referrers-tag").
				MatchKeyWords(append(discoverKeyWords(false, referrers...), foobar.Digest)...).
				Exec()
		})
	})
})

var _ = Describe("OCI image layout users:", func() {
	When("running discover command with json output", func() {
		format := "json"
		It("should discover direct referrers of a subject", func() {
			// prepare
			root := PrepareTempOCI(ArtifactRepo)
			subjectRef := LayoutRef(root, foobar.Tag)
			// test
			bytes := ORAS("discover", subjectRef, "-o", format, Flags.Layout).Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(index.Manifests).To(HaveLen(1))
			Expect(index.Manifests).Should(ContainElement(foobar.SBOMImageReferrer))
		})

		It("should discover matched referrer when filtering", func() {
			// prepare
			root := PrepareTempOCI(ArtifactRepo)
			subjectRef := LayoutRef(root, foobar.Tag)
			// test
			bytes := ORAS("discover", subjectRef, "-o", format, "--artifact-type", foobar.SBOMImageReferrer.ArtifactType, Flags.Layout).Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(index.Manifests).To(HaveLen(1))
			Expect(index.Manifests).Should(ContainElement(foobar.SBOMImageReferrer))
		})

		It("should discover no matched referrer", func() {
			// prepare
			root := PrepareTempOCI(ArtifactRepo)
			subjectRef := LayoutRef(root, foobar.Tag)
			// test
			bytes := ORAS("discover", subjectRef, "-o", format, "--artifact-type", "???", Flags.Layout).Exec().Out.Contents()
			var index ocispec.Index
			Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
			Expect(index.Manifests).To(HaveLen(0))
		})
	})

	When("running discover command with tree output", func() {
		format := "tree"
		referrers := []ocispec.Descriptor{foobar.SBOMImageReferrer, foobar.SignatureImageReferrer}
		It("should discover all referrers of a subject", func() {
			// prepare
			root := PrepareTempOCI(ArtifactRepo)
			subjectRef := LayoutRef(root, foobar.Tag)
			// test
			ORAS("discover", subjectRef, "-o", format, Flags.Layout).
				MatchKeyWords(append(discoverKeyWords(false, referrers...), LayoutRef(root, foobar.Digest))...).
				Exec()
		})

		It("should discover all referrers of a subject with annotations", func() {
			// prepare
			root := PrepareTempOCI(ArtifactRepo)
			subjectRef := LayoutRef(root, foobar.Tag)
			ORAS("discover", subjectRef, "-o", format, "-v", Flags.Layout).
				MatchKeyWords(append(discoverKeyWords(true, referrers...), LayoutRef(root, foobar.Digest))...).
				Exec()
		})
	})

	When("running discover command with table output", func() {
		format := "table"
		It("should get direct referrers of a subject", func() {
			referrers := []ocispec.Descriptor{foobar.SBOMImageReferrer}
			// prepare
			root := PrepareTempOCI(ArtifactRepo)
			subjectRef := LayoutRef(root, foobar.Tag)
			ORAS("discover", subjectRef, "-o", format, Flags.Layout).
				MatchKeyWords(append(discoverKeyWords(false, referrers...), foobar.Digest)...).
				Exec()
		})

		It("should discover no matched referrer", func() {
			// prepare
			root := PrepareTempOCI(ArtifactRepo)
			subjectRef := LayoutRef(root, foobar.Tag)
			out := ORAS("discover", subjectRef, "-o", format, "--artifact-type", "???", Flags.Layout).Exec().Out
			Expect(out).NotTo(gbytes.Say(foobar.SBOMImageReferrer.Digest.String()))
		})
	})
})
