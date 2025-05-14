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
			out := ORAS("discover", "--help").MatchKeyWords(feature.Preview.Mark+" Discover", feature.Preview.Description, ExampleDesc).Exec().Out
			gomega.Expect(out).Should(gbytes.Say("--distribution-spec string\\s+%s", regexp.QuoteMeta(feature.Preview.Mark)))
		})

		It("should say disable colors for --no-tty flag", func() {
			out := ORAS("discover", "--help").MatchKeyWords("disable colors").Exec().Out
			gomega.Expect(out).ShouldNot(gbytes.Say("disable progress bars"))
		})

		It("should show tree as default format type in help doc", func() {
			MatchDefaultFlagValue("format", "tree", "discover")
		})

		It("should show deprecation message when using table format", func() {
			ORAS("discover", RegistryRef(ZOTHost, ImageRepo, foobar.Tag), "--format", "table").MatchErrKeyWords(feature.DeprecationMessageTableFormat).Exec()
		})

		It("should fail when no subject reference provided", func() {
			ORAS("discover").ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})

		It("should fail when no tag or digest found in provided subject reference", func() {
			ORAS("discover", RegistryRef(ZOTHost, ImageRepo, "")).ExpectFailure().MatchErrKeyWords("Error:", "no tag or digest specified", "oras discover").Exec()
		})

		It("should fail with correct error message if both output and format flags are used", func() {
			ORAS("discover", RegistryRef(ZOTHost, ImageRepo, foobar.Tag), "--format", "json", "--output", "json").
				ExpectFailure().MatchErrKeyWords("--format", "--output", "same time").
				Exec()
		})

		It("should fail with correct error message if output flag is used before format flag", func() {
			ORAS("discover", RegistryRef(ZOTHost, ImageRepo, foobar.Tag), "--output", "json", "--format", "json").
				ExpectFailure().MatchErrKeyWords("--format", "--output", "same time").
				Exec()
		})

		It("should fail if invalid output type is used", func() {
			invalidType := "ukpkmkk"
			ORAS("discover", RegistryRef(ZOTHost, ImageRepo, foobar.Tag), "--output", invalidType).
				ExpectFailure().
				MatchErrKeyWords("Error:", "invalid format type", invalidType, "tree", "table", "json", "go-template").
				Exec()
		})

		It("should fail if given an invalid value for depth", func() {
			ORAS("discover", RegistryRef(ZOTHost, ImageRepo, foobar.Tag), "--depth", "0").
				ExpectFailure().
				MatchErrKeyWords("Error:", "depth value should be at least 1").
				Exec()
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
	type referrer struct {
		ocispec.Descriptor
		Referrers []ocispec.Descriptor
	}
	type subject struct {
		ocispec.Descriptor
		Referrers []referrer
	}
	subjectRef := RegistryRef(ZOTHost, ArtifactRepo, foobar.Tag)
	When("running discover command with json output", func() {
		format := "json"
		It("should discover referrers of a subject with deprecation hint", func() {
			bytes := ORAS("discover", subjectRef, "-o", format).MatchErrKeyWords(feature.Deprecated.Mark).Exec().Out.Contents()
			var subject subject
			Expect(json.Unmarshal(bytes, &subject)).ShouldNot(HaveOccurred())
			Expect(subject.Referrers).To(HaveLen(1))
			Expect(subject.Referrers[0].Descriptor).Should(Equal(foobar.SBOMImageReferrer))
		})
		It("should discover direct and indirect referrers of a subject by default", func() {
			bytes := ORAS("discover", subjectRef, "--format", format).Exec().Out.Contents()
			var subject subject
			// should show direct referrers correctly
			Expect(json.Unmarshal(bytes, &subject)).ShouldNot(HaveOccurred())
			Expect(subject.Referrers).To(HaveLen(1))
			Expect(subject.Referrers[0].Descriptor).Should(Equal(foobar.SBOMImageReferrer))
			// should show indirect referrers correctly
			referrer := subject.Referrers[0]
			Expect(referrer.Referrers).To(HaveLen(1))
			Expect(referrer.Referrers[0]).Should(Equal(foobar.SignatureImageReferrer))
		})
		It("should include information of subject and referrers manifests", func() {
			bytes := ORAS("discover", subjectRef, "--format", format).Exec().Out.Contents()
			var subject subject
			Expect(json.Unmarshal(bytes, &subject)).ShouldNot(HaveOccurred())
			Expect(subject.Descriptor).Should(Equal(foobar.FooBar))
			Expect(subject.Referrers).To(HaveLen(1))
			Expect(subject.Referrers[0].Descriptor).Should(Equal(foobar.SBOMImageReferrer))
		})

		It("should discover matched referrer when filtering", func() {
			bytes := ORAS("discover", subjectRef, "--format", format, "--artifact-type", foobar.SBOMImageReferrer.ArtifactType).
				Exec().Out.Contents()
			var subject subject
			Expect(json.Unmarshal(bytes, &subject)).ShouldNot(HaveOccurred())
			Expect(subject.Referrers).To(HaveLen(1))
			Expect(subject.Referrers[0].Descriptor).Should(Equal(foobar.SBOMImageReferrer))
		})

		It("should discover no matched referrer", func() {
			bytes := ORAS("discover", subjectRef, "--format", format, "--artifact-type", "???").Exec().Out.Contents()
			var subject subject
			Expect(json.Unmarshal(bytes, &subject)).ShouldNot(HaveOccurred())
			Expect(subject.Referrers).To(HaveLen(0))
		})

		It("should discover one referrer with matched platform", func() {
			bytes := ORAS("discover", RegistryRef(ZOTHost, ArtifactRepo, multi_arch.Tag), "--format", format, "--platform", "linux/amd64").
				Exec().Out.Contents()
			var subject subject
			Expect(json.Unmarshal(bytes, &subject)).ShouldNot(HaveOccurred())
			Expect(subject.Referrers).To(HaveLen(1))
			Expect(subject.Referrers[0].Descriptor).Should(Equal(multi_arch.LinuxAMD64Referrer))
		})

		It("should discover referrers correctly by depth 1", func() {
			bytes := ORAS("discover", subjectRef, "--format", format, "--depth", "1").Exec().Out.Contents()
			var subject subject
			// should show direct referrers correctly
			Expect(json.Unmarshal(bytes, &subject)).ShouldNot(HaveOccurred())
			Expect(subject.Referrers).To(HaveLen(1))
			Expect(subject.Referrers[0].Descriptor).Should(Equal(foobar.SBOMImageReferrer))
			// should not show indirect referrers
			referrer := subject.Referrers[0]
			Expect(referrer.Referrers).To(HaveLen(0))
		})

		It("should discover referrers correctly by depth 2", func() {
			bytes := ORAS("discover", subjectRef, "--format", format, "--depth", "2").Exec().Out.Contents()
			var subject subject
			// should show direct referrers correctly
			Expect(json.Unmarshal(bytes, &subject)).ShouldNot(HaveOccurred())
			Expect(subject.Referrers).To(HaveLen(1))
			Expect(subject.Referrers[0].Descriptor).Should(Equal(foobar.SBOMImageReferrer))
			// should show indirect referrers correctly
			referrer := subject.Referrers[0]
			Expect(referrer.Referrers).To(HaveLen(1))
			Expect(referrer.Referrers[0]).Should(Equal(foobar.SignatureImageReferrer))
		})

		It("should show the referrer field when no referrer is found", func() {
			bytes := ORAS("discover", RegistryRef(ZOTHost, ArtifactRepo, string(foobar.SignatureImageReferrer.Digest)), "--format", format).Exec().Out.Contents()
			var subject subject
			Expect(json.Unmarshal(bytes, &subject)).ShouldNot(HaveOccurred())
			Expect(subject.Referrers).ShouldNot(Equal(nil))
			Expect(subject.Referrers).To(HaveLen(0))
		})
	})

	When("running discover command with tree output", func() {
		referrers := []ocispec.Descriptor{foobar.SBOMImageReferrer, foobar.SignatureImageReferrer}
		directReferrers := foobar.SBOMImageReferrer
		indirectReferrers := foobar.SignatureImageReferrer
		It("should show as tree by default", func() {
			ORAS("discover", subjectRef).
				MatchKeyWords(append(discoverKeyWords(false, referrers...), RegistryRef(ZOTHost, ArtifactRepo, foobar.Digest))...).
				Exec()
		})
		format := "tree"
		It("should discover all referrers of a subject with deprecation hint", func() {
			ORAS("discover", subjectRef, "-o", format).
				MatchErrKeyWords(feature.Deprecated.Mark).
				MatchKeyWords(append(discoverKeyWords(false, referrers...), RegistryRef(ZOTHost, ArtifactRepo, foobar.Digest))...).
				Exec()
		})
		It("should discover all direct and indirect referrers of a subject by default", func() {
			err := ORAS("discover", subjectRef, "--format", format).
				MatchKeyWords(append(discoverKeyWords(false, referrers...), RegistryRef(ZOTHost, ArtifactRepo, foobar.Digest))...).
				Exec().Err
			Expect(err).NotTo(gbytes.Say(feature.Deprecated.Mark))
		})

		It("should discover all referrers of a subject via referrers API", func() {
			ORAS("discover", subjectRef, "--format", format, "--distribution-spec", "v1.1-referrers-api").
				MatchKeyWords(append(discoverKeyWords(false, referrers...), RegistryRef(ZOTHost, ArtifactRepo, foobar.Digest))...).
				Exec()
		})

		It("should discover all referrers of a subject with annotations", func() {
			ORAS("discover", subjectRef, "--format", format).
				MatchKeyWords(append(discoverKeyWords(true, referrers...), RegistryRef(ZOTHost, ArtifactRepo, foobar.Digest), "[annotations]")...).
				Exec()
		})

		It("should display <unknown> if a referrer has an empty artifact type", func() {
			ORAS("discover", RegistryRef(ZOTHost, ArtifactRepo, "multi"), "--format", format).
				MatchKeyWords("<unknown>").
				Exec()
		})

		It("should discover and display annotations with --verbose", func() {
			ORAS("discover", subjectRef, "--format", format, "-v").
				MatchKeyWords(append(discoverKeyWords(true, referrers...), RegistryRef(ZOTHost, ArtifactRepo, foobar.Digest), "[annotations]")...).
				Exec()
		})

		It("should not display annotations with --verbose=false", func() {
			referrers := []ocispec.Descriptor{foobar.SBOMImageReferrer, foobar.SignatureImageReferrer}
			out := ORAS("discover", subjectRef, "--format", format, "--verbose=false").
				MatchKeyWords(append(discoverKeyWords(false, referrers...), RegistryRef(ZOTHost, ArtifactRepo, foobar.Digest))...).
				Exec().Out
			Expect(out).NotTo(gbytes.Say("\\[annotations\\]"))
		})

		It("should show deprecation message when using verbose flag", func() {
			ORAS("discover", subjectRef, "--format", format, "--verbose").
				MatchErrKeyWords(feature.DeprecationMessageVerboseFlag).
				Exec()
		})

		It("should discover referrers correctly by depth 1", func() {
			out := ORAS("discover", subjectRef, "--format", format, "--depth", "1").
				MatchKeyWords(RegistryRef(ZOTHost, ArtifactRepo, foobar.Digest)).Exec().Out
			Expect(out).To(gbytes.Say(directReferrers.Digest.String()))
			Expect(out).NotTo(gbytes.Say(indirectReferrers.Digest.String()))
		})

		It("should discover referrers correctly by depth 2", func() {
			out := ORAS("discover", subjectRef, "--format", format, "--depth", "2").
				MatchKeyWords(RegistryRef(ZOTHost, ArtifactRepo, foobar.Digest)).Exec().Out
			Expect(out).To(gbytes.Say(directReferrers.Digest.String()))
			Expect(out).To(gbytes.Say(indirectReferrers.Digest.String()))
		})
	})
	When("running discover command with table output", func() {
		format := "table"
		It("should show all referrers of a subject", func() {
			referrers := []ocispec.Descriptor{foobar.SBOMImageReferrer, foobar.SBOMImageReferrer}
			ORAS("discover", subjectRef, "--format", format, "--depth", "1").
				MatchKeyWords(append(discoverKeyWords(false, referrers...), foobar.Digest)...).
				Exec()
		})
	})
	When("running discover command with go-template output", func() {
		It("should show referrers digest of a subject", func() {
			ORAS("discover", subjectRef, "--format", "go-template={{(first .referrers).reference}}").
				MatchContent(RegistryRef(ZOTHost, ArtifactRepo, foobar.SBOMImageReferrer.Digest.String())).
				Exec()
		})
	})
})

var _ = Describe("1.0 registry users:", func() {
	type referrer struct {
		ocispec.Descriptor
		Referrers []ocispec.Descriptor
	}
	type subject struct {
		ocispec.Descriptor
		Referrers []referrer
	}
	subjectRef := RegistryRef(FallbackHost, ArtifactRepo, foobar.Tag)
	When("running discover command", func() {
		It("should discover all direct and indirect referrers of a subject by default via json output", func() {
			bytes := ORAS("discover", subjectRef, "--format", "json").Exec().Out.Contents()
			var subject subject
			// should show direct referrers correctly
			Expect(json.Unmarshal(bytes, &subject)).ShouldNot(HaveOccurred())
			Expect(subject.Referrers).To(HaveLen(1))
			Expect(subject.Referrers[0].Descriptor).Should(Equal(foobar.SBOMImageReferrer))
			// should show indirect referrers correctly
			referrer := subject.Referrers[0]
			Expect(referrer.Referrers).To(HaveLen(1))
			Expect(referrer.Referrers[0]).Should(Equal(foobar.SignatureImageReferrer))
		})

		It("should include information of subject and referrers manifests via json output", func() {
			bytes := ORAS("discover", subjectRef, "--format", "json").Exec().Out.Contents()
			var subject subject
			Expect(json.Unmarshal(bytes, &subject)).ShouldNot(HaveOccurred())
			Expect(subject.Descriptor).Should(Equal(foobar.FooBar))
			Expect(subject.Referrers).To(HaveLen(1))
			Expect(subject.Referrers[0].Descriptor).Should(Equal(foobar.SBOMImageReferrer))
		})

		It("should discover matched referrer when filtering via json output", func() {
			bytes := ORAS("discover", subjectRef, "--format", "json", "--artifact-type", foobar.SBOMImageReferrer.ArtifactType).Exec().Out.Contents()
			var subject subject
			Expect(json.Unmarshal(bytes, &subject)).ShouldNot(HaveOccurred())
			Expect(subject.Referrers).To(HaveLen(1))
			Expect(subject.Referrers[0].Descriptor).Should(Equal(foobar.SBOMImageReferrer))
		})

		It("should discover no referrer when not matching via json output", func() {
			bytes := ORAS("discover", subjectRef, "--format", "json", "--artifact-type", "???").Exec().Out.Contents()
			var subject subject
			Expect(json.Unmarshal(bytes, &subject)).ShouldNot(HaveOccurred())
			Expect(subject.Referrers).To(HaveLen(0))
		})

		It("should discover referrers correctly by depth 1 via json output", func() {
			bytes := ORAS("discover", subjectRef, "--format", "json", "--depth", "1").Exec().Out.Contents()
			var subject subject
			// should show direct referrers correctly
			Expect(json.Unmarshal(bytes, &subject)).ShouldNot(HaveOccurred())
			Expect(subject.Referrers).To(HaveLen(1))
			Expect(subject.Referrers[0].Descriptor).Should(Equal(foobar.SBOMImageReferrer))
			// should not show indirect referrers
			referrer := subject.Referrers[0]
			Expect(referrer.Referrers).To(HaveLen(0))
		})

		It("should discover referrers correctly by depth 2", func() {
			bytes := ORAS("discover", subjectRef, "--format", "json", "--depth", "2").Exec().Out.Contents()
			var subject subject
			// should show direct referrers correctly
			Expect(json.Unmarshal(bytes, &subject)).ShouldNot(HaveOccurred())
			Expect(subject.Referrers).To(HaveLen(1))
			Expect(subject.Referrers[0].Descriptor).Should(Equal(foobar.SBOMImageReferrer))
			// should show indirect referrers correctly
			referrer := subject.Referrers[0]
			Expect(referrer.Referrers).To(HaveLen(1))
			Expect(referrer.Referrers[0]).Should(Equal(foobar.SignatureImageReferrer))
		})

		It("should show the referrer field when no referrer is found", func() {
			bytes := ORAS("discover", RegistryRef(ZOTHost, ArtifactRepo, string(foobar.SignatureImageReferrer.Digest)), "--format", "json").Exec().Out.Contents()
			var subject subject
			Expect(json.Unmarshal(bytes, &subject)).ShouldNot(HaveOccurred())
			Expect(subject.Referrers).ShouldNot(Equal(nil))
			Expect(subject.Referrers).To(HaveLen(0))
		})

		It("should discover all direct and indirect referrers of a subject by default via tree output", func() {
			referrers := []ocispec.Descriptor{foobar.SBOMImageReferrer, foobar.SignatureImageReferrer}
			ORAS("discover", subjectRef, "--format", "tree").
				MatchKeyWords(append(discoverKeyWords(false, referrers...), RegistryRef(FallbackHost, ArtifactRepo, foobar.Digest))...).
				Exec()
		})

		It("should discover referrers correctly by depth 1 via tree output", func() {
			out := ORAS("discover", subjectRef, "--format", "tree", "--depth", "1").Exec().Out
			Expect(out).To(gbytes.Say(foobar.SBOMImageReferrer.Digest.String()))
			Expect(out).NotTo(gbytes.Say(foobar.SignatureImageReferrer.Digest.String()))
		})

		It("should discover referrers correctly by depth 2", func() {
			out := ORAS("discover", subjectRef, "--format", "tree", "--depth", "2").Exec().Out
			Expect(out).To(gbytes.Say(foobar.SBOMImageReferrer.Digest.String()))
			Expect(out).To(gbytes.Say(foobar.SignatureImageReferrer.Digest.String()))
		})

		It("should discover all referrers with annotation via tree output", func() {
			referrers := []ocispec.Descriptor{foobar.SBOMImageReferrer, foobar.SignatureImageReferrer}
			ORAS("discover", subjectRef, "--format", "tree", "-v").
				MatchKeyWords(append(discoverKeyWords(true, referrers...), RegistryRef(FallbackHost, ArtifactRepo, foobar.Digest))...).
				Exec()
		})

		It("should discover direct referrers of a subject via table output", func() {
			referrers := []ocispec.Descriptor{foobar.SBOMImageReferrer}
			ORAS("discover", subjectRef, "--format", "table", "--depth", "1").
				MatchKeyWords(append(discoverKeyWords(false, referrers...), foobar.Digest)...).
				Exec()
		})

		It("should discover direct referrers explicitly via tag scheme", func() {
			referrers := []ocispec.Descriptor{foobar.SBOMImageReferrer}
			ORAS("discover", subjectRef, "--format", "table", "--distribution-spec", "v1.1-referrers-tag", "--depth", "1").
				MatchKeyWords(append(discoverKeyWords(false, referrers...), foobar.Digest)...).
				Exec()
		})
	})
})

var _ = Describe("OCI image layout users:", func() {
	When("running discover command with json output", func() {
		type referrer struct {
			ocispec.Descriptor
			Referrers []ocispec.Descriptor
		}
		type subject struct {
			ocispec.Descriptor
			Referrers []referrer
		}
		format := "json"
		It("should discover direct and indirect referrers of a subject by default", func() {
			// prepare
			root := PrepareTempOCI(ArtifactRepo)
			subjectRef := LayoutRef(root, foobar.Tag)
			// test
			bytes := ORAS("discover", subjectRef, "--format", format, Flags.Layout).Exec().Out.Contents()
			var subject subject
			// should show direct referrers correctly
			Expect(json.Unmarshal(bytes, &subject)).ShouldNot(HaveOccurred())
			Expect(subject.Referrers).To(HaveLen(1))
			Expect(subject.Referrers[0].Descriptor).Should(Equal(foobar.SBOMImageReferrer))
			// should show indirect referrers correctly
			referrer := subject.Referrers[0]
			Expect(referrer.Referrers).To(HaveLen(1))
			Expect(referrer.Referrers[0]).Should(Equal(foobar.SignatureImageReferrer))
		})

		It("should discover referrers correctly by depth 1", func() {
			// prepare
			root := PrepareTempOCI(ArtifactRepo)
			subjectRef := LayoutRef(root, foobar.Tag)
			// test
			bytes := ORAS("discover", subjectRef, "--format", format, Flags.Layout, "--depth", "1").Exec().Out.Contents()
			var subject subject
			// should show direct referrers correctly
			Expect(json.Unmarshal(bytes, &subject)).ShouldNot(HaveOccurred())
			Expect(subject.Referrers).To(HaveLen(1))
			Expect(subject.Referrers[0].Descriptor).Should(Equal(foobar.SBOMImageReferrer))
			// should not show indirect referrers
			referrer := subject.Referrers[0]
			Expect(referrer.Referrers).To(HaveLen(0))
		})

		It("should discover referrers correctly by depth 2", func() {
			// prepare
			root := PrepareTempOCI(ArtifactRepo)
			subjectRef := LayoutRef(root, foobar.Tag)
			// test
			bytes := ORAS("discover", subjectRef, "--format", format, Flags.Layout, "--depth", "2").Exec().Out.Contents()
			var subject subject
			// should show direct referrers correctly
			Expect(json.Unmarshal(bytes, &subject)).ShouldNot(HaveOccurred())
			Expect(subject.Referrers).To(HaveLen(1))
			Expect(subject.Referrers[0].Descriptor).Should(Equal(foobar.SBOMImageReferrer))
			// should show indirect referrers correctly
			referrer := subject.Referrers[0]
			Expect(referrer.Referrers).To(HaveLen(1))
			Expect(referrer.Referrers[0]).Should(Equal(foobar.SignatureImageReferrer))
		})

		It("should include information of subject and referrers manifests", func() {
			// prepare
			root := PrepareTempOCI(ArtifactRepo)
			subjectRef := LayoutRef(root, foobar.Tag)
			// test
			bytes := ORAS("discover", subjectRef, "--format", format, Flags.Layout).Exec().Out.Contents()
			var subject subject
			Expect(json.Unmarshal(bytes, &subject)).ShouldNot(HaveOccurred())
			Expect(subject.Descriptor).Should(Equal(foobar.FooBarOCI))
			Expect(subject.Referrers).To(HaveLen(1))
			Expect(subject.Referrers[0].Descriptor).Should(Equal(foobar.SBOMImageReferrer))
		})

		It("should discover matched referrer when filtering", func() {
			// prepare
			root := PrepareTempOCI(ArtifactRepo)
			subjectRef := LayoutRef(root, foobar.Tag)
			// test
			bytes := ORAS("discover", subjectRef, "--format", format, "--artifact-type", foobar.SBOMImageReferrer.ArtifactType, Flags.Layout).Exec().Out.Contents()
			var disv subject
			Expect(json.Unmarshal(bytes, &disv)).ShouldNot(HaveOccurred())
			Expect(disv.Referrers).To(HaveLen(1))
			Expect(disv.Referrers[0].Descriptor).Should(Equal(foobar.SBOMImageReferrer))
		})

		It("should discover no matched referrer", func() {
			// prepare
			root := PrepareTempOCI(ArtifactRepo)
			subjectRef := LayoutRef(root, foobar.Tag)
			// test
			bytes := ORAS("discover", subjectRef, "--format", format, "--artifact-type", "???", Flags.Layout).Exec().Out.Contents()
			var disv subject
			Expect(json.Unmarshal(bytes, &disv)).ShouldNot(HaveOccurred())
			Expect(disv.Referrers).To(HaveLen(0))
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
			ORAS("discover", subjectRef, "--format", format, Flags.Layout).
				MatchKeyWords(append(discoverKeyWords(false, referrers...), LayoutRef(root, foobar.Digest))...).
				Exec()
		})

		It("should discover referrers correctly by depth 1", func() {
			// prepare
			root := PrepareTempOCI(ArtifactRepo)
			subjectRef := LayoutRef(root, foobar.Tag)
			// test
			out := ORAS("discover", subjectRef, "--format", format, Flags.Layout, "--depth", "1").Exec().Out
			Expect(out).To(gbytes.Say(foobar.SBOMImageReferrer.Digest.String()))
			Expect(out).NotTo(gbytes.Say(foobar.SignatureImageReferrer.Digest.String()))
		})

		It("should discover referrers correctly by depth 2", func() {
			// prepare
			root := PrepareTempOCI(ArtifactRepo)
			subjectRef := LayoutRef(root, foobar.Tag)
			// test
			out := ORAS("discover", subjectRef, "--format", format, Flags.Layout, "--depth", "2").Exec().Out
			Expect(out).To(gbytes.Say(foobar.SBOMImageReferrer.Digest.String()))
			Expect(out).To(gbytes.Say(foobar.SignatureImageReferrer.Digest.String()))
		})

		It("should discover all referrers of a subject with annotations", func() {
			// prepare
			root := PrepareTempOCI(ArtifactRepo)
			subjectRef := LayoutRef(root, foobar.Tag)
			ORAS("discover", subjectRef, "--format", format, "-v", Flags.Layout).
				MatchKeyWords(append(discoverKeyWords(true, referrers...), LayoutRef(root, foobar.Digest), "[annotations]")...).
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
			ORAS("discover", subjectRef, "--format", format, Flags.Layout, "--depth", "1").
				MatchKeyWords(append(discoverKeyWords(false, referrers...), foobar.Digest)...).
				Exec()
		})

		It("should discover no matched referrer", func() {
			root := PrepareTempOCI(ArtifactRepo)
			subjectRef := LayoutRef(root, foobar.Tag)
			out := ORAS("discover", subjectRef, "--format", format, "--artifact-type", "???", Flags.Layout).Exec().Out
			Expect(out).NotTo(gbytes.Say(foobar.SBOMImageReferrer.Digest.String()))
		})
	})
})
