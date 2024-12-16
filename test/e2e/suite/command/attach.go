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
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"oras.land/oras/test/e2e/internal/testdata/feature"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	"oras.land/oras/test/e2e/internal/testdata/multi_arch"
	. "oras.land/oras/test/e2e/internal/utils"
	"oras.land/oras/test/e2e/internal/utils/match"
)

func attachTestRepo(text string) string {
	return fmt.Sprintf("command/attach/%d/%s", GinkgoRandomSeed(), text)
}

var _ = Describe("ORAS beginners:", func() {
	When("running attach command", func() {
		RunAndShowPreviewInHelp([]string{"attach"})

		It("should show preview and help doc", func() {
			out := ORAS("attach", "--help").MatchKeyWords(feature.Preview.Mark+" Attach", feature.Preview.Description, ExampleDesc).Exec().Out
			gomega.Expect(out).Should(gbytes.Say("--distribution-spec string\\s+%s", regexp.QuoteMeta(feature.Preview.Mark)))
		})

		It("should not show --verbose in help doc", func() {
			out := ORAS("push", "--help").MatchKeyWords(ExampleDesc).Exec().Out
			gomega.Expect(out).ShouldNot(gbytes.Say("--verbose"))
		})

		It("should show deprecation message and print unnamed status output for --verbose", func() {
			testRepo := attachTestRepo("test-verbose")
			CopyZOTRepo(ImageRepo, testRepo)
			subjectRef := RegistryRef(ZOTHost, testRepo, foobar.Tag)
			stateKeys := []match.StateKey{
				foobar.AttachFileStateKey,
				{Digest: "44136fa355b3", Name: "application/vnd.oci.empty.v1+json"},
			}
			ORAS("attach", "--artifact-type", "test/attach", "--verbose", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia)).
				WithWorkDir(PrepareTempFiles()).
				MatchErrKeyWords(feature.DeprecationMessageVerboseFlag).
				MatchStatus(stateKeys, true, len(stateKeys)).Exec()
		})

		It("should show deprecation message and should NOT print unnamed status output for --verbose=false", func() {
			testRepo := attachTestRepo("test-verbose-false")
			CopyZOTRepo(ImageRepo, testRepo)
			subjectRef := RegistryRef(ZOTHost, testRepo, foobar.Tag)
			stateKeys := []match.StateKey{foobar.AttachFileStateKey}
			out := ORAS("attach", "--artifact-type", "test/attach", "--verbose=false", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia)).
				WithWorkDir(PrepareTempFiles()).
				MatchErrKeyWords(feature.DeprecationMessageVerboseFlag).
				MatchStatus(stateKeys, false, len(stateKeys)).Exec().Out
			gomega.Expect(out).ShouldNot(gbytes.Say("application/vnd.oci.empty.v1+json"))
		})

		It("should show text as default format type in help doc", func() {
			MatchDefaultFlagValue("format", "text", "attach")
		})

		It("should fail when no subject reference provided", func() {
			ORAS("attach", "--artifact-type", "oras/test").ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})

		It("should fail if no file reference or manifest annotation provided for registry", func() {
			ORAS("attach", "--artifact-type", "oras/test", RegistryRef(ZOTHost, ImageRepo, foobar.Tag)).
				ExpectFailure().MatchErrKeyWords("Error: neither file nor annotation", "Usage:").Exec()
		})

		It("should fail if no file reference or manifest annotation provided for OCI layout", func() {
			root := GinkgoT().TempDir()
			ORAS("attach", "--artifact-type", "oras/test", Flags.Layout, LayoutRef(root, foobar.Tag)).
				ExpectFailure().MatchErrKeyWords("Error: neither file nor annotation", "Usage:").Exec()
		})

		It("should fail if distribution spec is unknown", func() {
			ORAS("attach", "--artifact-type", "oras/test", RegistryRef(ZOTHost, ImageRepo, foobar.Tag), "--distribution-spec", "???").
				ExpectFailure().MatchErrKeyWords("unknown distribution specification flag").Exec()
		})

		It("should fail with error suggesting subject missed", func() {
			err := ORAS("attach", "--artifact-type", "oras/test", RegistryRef(ZOTHost, ImageRepo, "")).ExpectFailure().Exec().Err
			Expect(err).Should(gbytes.Say("Error"))
			Expect(err).Should(gbytes.Say("\nAre you missing an artifact reference to attach to?"))
		})

		It("should fail with error suggesting right form", func() {
			err := ORAS("attach", "--artifact-type", "oras/test", RegistryRef(ZOTHost, ImageRepo, ""), "./test.json").ExpectFailure().Exec().Err
			Expect(err).Should(gbytes.Say("Error"))
			Expect(err).Should(gbytes.Say("no tag or digest specified"))
			Expect(err).ShouldNot(gbytes.Say("\nAre you missing an artifact reference to attach to?"))
		})

		It("should fail and show detailed error description if no argument provided", func() {
			err := ORAS("attach").ExpectFailure().Exec().Err
			Expect(err).Should(gbytes.Say("Error"))
			Expect(err).Should(gbytes.Say("\nUsage: oras attach"))
			Expect(err).Should(gbytes.Say("\n"))
			Expect(err).Should(gbytes.Say(`Run "oras attach -h"`))
		})

		It("should fail if distribution spec is not valid", func() {
			testRepo := attachTestRepo("invalid-image-spec")
			CopyZOTRepo(ImageRepo, testRepo)
			subjectRef := RegistryRef(ZOTHost, testRepo, foobar.Tag)
			invalidFlag := "???"
			ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), Flags.DistributionSpec, invalidFlag).
				ExpectFailure().
				WithWorkDir(PrepareTempFiles()).
				MatchErrKeyWords("Error:", invalidFlag, "Available options: v1.1-referrers-tag, v1.1-referrers-api").
				Exec()
		})
	})
})

var _ = Describe("1.1 registry users:", func() {
	When("running attach command", func() {
		It("should attach a file to a subject and output status", func() {
			testRepo := attachTestRepo("attach-tag")
			CopyZOTRepo(ImageRepo, testRepo)
			subjectRef := RegistryRef(ZOTHost, testRepo, foobar.Tag)
			ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia)).
				WithWorkDir(PrepareTempFiles()).
				MatchKeyWords(fmt.Sprintf("Attached to [registry] %s", RegistryRef(ZOTHost, testRepo, foobar.Digest))).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()
		})

		It("should attach a file to a subject and output status", func() {
			testRepo := attachTestRepo("attach-digest")
			CopyZOTRepo(ImageRepo, testRepo)
			subjectRef := RegistryRef(ZOTHost, testRepo, foobar.Digest)
			ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia)).
				WithWorkDir(PrepareTempFiles()).
				MatchKeyWords(fmt.Sprintf("Attached to [registry] %s", subjectRef)).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()
		})

		It("should attach a file to an arch-specific subject", func() {
			// prepare
			testRepo := attachTestRepo("arch-specific")
			CopyZOTRepo(ImageRepo, testRepo)
			// test
			subjectRef := RegistryRef(ZOTHost, testRepo, multi_arch.Tag)
			artifactType := "test/attach"
			out := ORAS("attach", "--artifact-type", artifactType, subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--format", "go-template={{.digest}}", "--platform", "linux/amd64").
				WithWorkDir(PrepareTempFiles()).Exec().Out.Contents()
			// validate
			ORAS("discover", "--artifact-type", artifactType, RegistryRef(ZOTHost, testRepo, multi_arch.LinuxAMD64.Digest.String())).MatchKeyWords(string(out)).Exec()
		})

		It("should attach a file to a subject and export the built manifest", func() {
			// prepare
			testRepo := attachTestRepo("export-manifest")
			tempDir := PrepareTempFiles()
			exportName := "manifest.json"
			subjectRef := RegistryRef(ZOTHost, testRepo, foobar.Tag)
			CopyZOTRepo(ImageRepo, testRepo)
			// test
			ref := ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--export-manifest", exportName, "--format", "go-template={{.reference}}").
				WithWorkDir(tempDir).Exec().Out.Contents()
			// validate
			fetched := ORAS("manifest", "fetch", string(ref)).Exec().Out.Contents()
			MatchFile(filepath.Join(tempDir, exportName), string(fetched), DefaultTimeout)
		})

		It("should attach a file to a subject and format the digest reference", func() {
			// prepare
			testRepo := attachTestRepo("format-ref")
			tempDir := PrepareTempFiles()
			exportName := "manifest.json"
			subjectRef := RegistryRef(ZOTHost, testRepo, foobar.Tag)
			CopyZOTRepo(ImageRepo, testRepo)
			// test
			delimitter := "---"
			output := ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--export-manifest", exportName, "--format", fmt.Sprintf("go-template={{.reference}}%s{{.artifactType}}", delimitter)).
				WithWorkDir(tempDir).Exec().Out.Contents()
			ref, artifactType, _ := strings.Cut(string(output), delimitter)
			// validate
			Expect(artifactType).To(Equal("test/attach"))
			fetched := ORAS("manifest", "fetch", ref).Exec().Out.Contents()
			MatchFile(filepath.Join(tempDir, exportName), string(fetched), DefaultTimeout)
		})

		It("should attach a file to a subject and format json", func() {
			// prepare
			testRepo := attachTestRepo("format-json")
			tempDir := PrepareTempFiles()
			exportName := "manifest.json"
			subjectRef := RegistryRef(ZOTHost, testRepo, foobar.Tag)
			CopyZOTRepo(ImageRepo, testRepo)
			// test
			out := ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--export-manifest", exportName, "--format", "json").
				WithWorkDir(tempDir).Exec().Out
			// validate
			Expect(out).To(gbytes.Say(RegistryRef(ZOTHost, testRepo, "")))
		})

		It("should attach a file to a subject twice", func() {
			// prepare
			testRepo := attachTestRepo("attach-twice")
			tempDir := PrepareTempFiles()
			subjectRef := RegistryRef(ZOTHost, testRepo, foobar.Tag)
			CopyZOTRepo(ImageRepo, testRepo)
			// test
			ref1 := ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--format", "go-template={{.reference}}").
				WithWorkDir(tempDir).Exec().Out.Contents()
			ref2 := ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--format", "go-template={{.reference}}").
				WithWorkDir(tempDir).Exec().Out.Contents()
			// validate
			ORAS("discover", subjectRef, "--format", "go-template={{range .manifests}}{{println .reference}}{{end}}").MatchKeyWords(string(ref1), string(ref2)).Exec()
		})

		It("should attach a file via a OCI Image", func() {
			testRepo := attachTestRepo("image")
			tempDir := PrepareTempFiles()
			subjectRef := RegistryRef(ZOTHost, testRepo, foobar.Tag)
			CopyZOTRepo(ImageRepo, testRepo)
			// test
			ref := ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--format", "go-template={{.reference}}").
				WithWorkDir(tempDir).Exec().Out.Contents()
			// validate
			out := ORAS("discover", subjectRef, "--format", "go-template={{range .manifests}}{{println .reference}}{{end}}").Exec().Out
			Expect(out).To(gbytes.Say(string(ref)))
		})

		It("should attach file with path validation disabled", func() {
			testRepo := attachTestRepo("simple")
			absAttachFileName := filepath.Join(PrepareTempFiles(), foobar.AttachFileName)

			subjectRef := RegistryRef(ZOTHost, testRepo, foobar.Tag)
			CopyZOTRepo(ImageRepo, testRepo)
			statusKey := foobar.AttachFileStateKey
			statusKey.Name = absAttachFileName
			ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", absAttachFileName, foobar.AttachFileMedia), "--disable-path-validation").Exec()
		})

		It("should fail path validation when attaching file with absolute path", func() {
			testRepo := attachTestRepo("simple")
			absAttachFileName := filepath.Join(PrepareTempFiles(), foobar.AttachFileName)

			subjectRef := RegistryRef(ZOTHost, testRepo, foobar.Tag)
			CopyZOTRepo(ImageRepo, testRepo)
			statusKey := foobar.AttachFileStateKey
			statusKey.Name = absAttachFileName
			ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", absAttachFileName, foobar.AttachFileMedia)).
				ExpectFailure().
				Exec()
		})
	})
})

var _ = Describe("1.0 registry users:", func() {
	When("running attach command", func() {
		It("should attach a file via a OCI Image", func() {
			testRepo := attachTestRepo("fallback/image")
			tempDir := PrepareTempFiles()
			subjectRef := RegistryRef(FallbackHost, testRepo, foobar.Tag)
			prepare(RegistryRef(FallbackHost, ArtifactRepo, foobar.Tag), subjectRef)
			// test
			ref := ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--format", "go-template={{.reference}}").
				WithWorkDir(tempDir).Exec().Out.Contents()
			// validate
			out := ORAS("discover", subjectRef, "--format", "go-template={{range .manifests}}{{println .reference}}{{end}}").Exec().Out
			Expect(out).To(gbytes.Say(string(ref)))
		})

		It("should attach a file via a OCI Image by default", func() {
			testRepo := attachTestRepo("fallback/default")
			tempDir := PrepareTempFiles()
			subjectRef := RegistryRef(FallbackHost, testRepo, foobar.Tag)
			prepare(RegistryRef(FallbackHost, ArtifactRepo, foobar.Tag), subjectRef)
			// test
			ref := ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--format", "go-template={{.reference}}").
				WithWorkDir(tempDir).Exec().Out.Contents()

			// validate
			out := ORAS("discover", subjectRef, "--format", "go-template={{range .manifests}}{{println .reference}}{{end}}").Exec().Out
			Expect(out).To(gbytes.Say(string(ref)))
		})

		It("should attach a file via a OCI Image and generate referrer via tag schema", func() {
			testRepo := attachTestRepo("fallback/tag_schema")
			tempDir := PrepareTempFiles()
			subjectRef := RegistryRef(FallbackHost, testRepo, foobar.Tag)
			prepare(RegistryRef(FallbackHost, ArtifactRepo, foobar.Tag), subjectRef)
			// test
			ref := ORAS("attach", "--artifact-type", "test/attach", subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--distribution-spec", "v1.1-referrers-tag", "--format", "go-template={{.reference}}").
				WithWorkDir(tempDir).Exec().Out.Contents()

			// validate
			out := ORAS("discover", subjectRef, "--format", "go-template={{range .manifests}}{{println .reference}}{{end}}").Exec().Out
			Expect(out).To(gbytes.Say(string(ref)))
		})
	})
})

var _ = Describe("OCI image layout users:", func() {
	When("running attach command", func() {
		It("should attach a file to a subject", func() {
			root := PrepareTempOCI(ImageRepo)
			subjectRef := LayoutRef(root, foobar.Tag)
			ORAS("attach", "--artifact-type", "test/attach", Flags.Layout, subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia)).
				WithWorkDir(root).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()
		})

		It("should attach a file to a subject and export the built manifest", func() {
			// prepare
			exportName := "manifest.json"
			root := PrepareTempOCI(ImageRepo)
			subjectRef := LayoutRef(root, foobar.Tag)
			// test
			ref := ORAS("attach", "--artifact-type", "test/attach", Flags.Layout, subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--export-manifest", exportName, "--format", "go-template={{.reference}}").
				WithWorkDir(root).Exec().Out.Contents()
			// validate
			fetched := ORAS("manifest", "fetch", Flags.Layout, string(ref)).Exec().Out.Contents()
			MatchFile(filepath.Join(root, exportName), string(fetched), DefaultTimeout)
		})

		It("should attach a file to an arch-specific subject", func() {
			root := PrepareTempOCI(ImageRepo)
			subjectRef := LayoutRef(root, multi_arch.Tag)
			artifactType := "test/attach"
			// test
			out := ORAS("attach", Flags.Layout, "--artifact-type", artifactType, subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--format", "go-template={{.digest}}", "--platform", "linux/amd64").
				WithWorkDir(PrepareTempFiles()).Exec().Out.Contents()
			// validate
			ORAS("discover", Flags.Layout, "--artifact-type", artifactType, LayoutRef(root, multi_arch.LinuxAMD64.Digest.String())).MatchKeyWords(string(out)).Exec()
		})

		It("should attach a file via a OCI Image", func() {
			root := PrepareTempOCI(ImageRepo)
			subjectRef := LayoutRef(root, foobar.Tag)
			// test
			ref := ORAS("attach", "--artifact-type", "test/attach", Flags.Layout, subjectRef, fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia), "--format", "go-template={{.reference}}").
				WithWorkDir(root).Exec().Out.Contents()
			// validate
			out := ORAS("discover", Flags.Layout, subjectRef, "--format", "go-template={{range .manifests}}{{println .reference}}{{end}}").Exec().Out
			Expect(out).To(gbytes.Say(string(ref)))
		})
	})
})
