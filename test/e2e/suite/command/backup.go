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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/oras-project/oras-go/v3"
	"oras.land/oras/test/e2e/internal/testdata/feature"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	ma "oras.land/oras/test/e2e/internal/testdata/multi_arch"
	. "oras.land/oras/test/e2e/internal/utils"
	"oras.land/oras/test/e2e/internal/utils/match"
)

func verifyBackupDirectoryStructure(backupPath string) {
	Expect(backupPath).To(BeADirectory())
	Expect(filepath.Join(backupPath, "ingest")).ToNot(BeAnExistingFile())
}

func verifyBackupTarStructure(tarPath string) {
	Expect(tarPath).To(BeAnExistingFile())
	Expect(tarPath).NotTo(BeADirectory())
}

// ociLayoutPath is the flag that reads an OCI layout at a path, taking the
// reference as a separate argument.
const ociLayoutPath = "--oci-layout-path"

// compareBackupRef compares the manifest of srcRef with the manifest stored
// under layoutRef in the backup at layoutPath. Backups store tagged artifacts
// under their full reference (e.g. localhost:5000/repo:v1), which the
// --oci-layout <path>:<tag> form cannot address, so the backup is read with
// --oci-layout-path. layoutRef is a full reference or a digest.
func compareBackupRef(srcRef, layoutPath, layoutRef string) {
	srcManifest := ORAS("manifest", "fetch", srcRef).WithDescription("fetch from source to validate").Exec().Out.Contents()
	dstManifest := ORAS("manifest", "fetch", ociLayoutPath, layoutPath, layoutRef).WithDescription("fetch from destination OCI layout to validate").Exec().Out.Contents()
	Expect(srcManifest).To(Equal(dstManifest))
}

// backupReferrers returns the digests of the referrers of layoutRef in the
// backup at layoutPath.
func backupReferrers(layoutPath, layoutRef string) []string {
	referrers := ORAS("discover", ociLayoutPath, layoutPath, layoutRef, "--format", "go-template={{range .referrers}}{{println .digest}}{{end}}").Exec().Out.Contents()
	return strings.Split(strings.TrimSpace(string(referrers)), "\n")
}

func backupTestRepo(text string) string {
	return fmt.Sprintf("command/backup/%d/%s", GinkgoRandomSeed(), text)
}

var _ = Describe("ORAS beginners:", func() {
	When("running backup command", func() {
		It("should show help description with experimental flag", func() {
			out := ORAS("backup", "--help").MatchKeyWords(ExampleDesc).Exec().Out
			gomega.Expect(out).Should(gbytes.Say(regexp.QuoteMeta(feature.Experimental.Mark)))
		})

		It("should fail when no reference provided", func() {
			ORAS("backup").ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})

		It("should fail when no output path provided", func() {
			ORAS("backup", RegistryRef(ZOTHost, ImageRepo, foobar.Tag)).ExpectFailure().
				MatchErrKeyWords("Error:", `required flag(s) "output" not set`).Exec()
		})

		It("should fail when the output path is empty", func() {
			ORAS("backup", RegistryRef(ZOTHost, ImageRepo, foobar.Tag), "-o", "").ExpectFailure().
				MatchErrKeyWords("Error:", `the output path cannot be empty`).Exec()
		})

		It("should fail when source doesn't exist", func() {
			tmpDir := GinkgoT().TempDir()
			defer os.RemoveAll(tmpDir)

			ORAS("backup", "--output", tmpDir, RegistryRef(ZOTHost, ImageRepo, InvalidTag)).ExpectFailure().
				MatchErrKeyWords("Error:", InvalidTag).Exec()
		})

		It("should fail with appropriate error when digest is provided", func() {
			tmpDir := GinkgoT().TempDir()
			outDir := filepath.Join(tmpDir, "digest-reference")

			// Test with a malformed reference
			ORAS("backup", "--output", outDir, "invalid/format@sha256:digest").ExpectFailure().
				MatchErrKeyWords("Error:", "digest references are not supported").Exec()
		})

		It("should fail with appropriate error when invalid tag format provided", func() {
			tmpDir := GinkgoT().TempDir()
			outDir := filepath.Join(tmpDir, "invalid-tag")

			// Test with invalid tag format
			ORAS("backup", "--output", outDir, "localhost:5000/repo:invalid+tag").ExpectFailure().
				MatchErrKeyWords("Error:").Exec()
		})

		It("should fail with appropriate error when tag is provided with digest", func() {
			tmpDir := GinkgoT().TempDir()
			outDir := filepath.Join(tmpDir, "invalid-reference")

			// Test with a malformed reference
			ORAS("backup", "--output", outDir, "invalid/format:v1,@sha256:123abc").ExpectFailure().
				MatchErrKeyWords("Error:", "digest references are not supported").Exec()
		})

		It("should fail if given an invalid value for concurrency", func() {
			tmpDir := GinkgoT().TempDir()
			outDir := filepath.Join(tmpDir, "invalid-concurrency")
			for _, val := range []string{"0", "-1"} {
				ORAS("backup", "--output", outDir, RegistryRef(ZOTHost, ImageRepo, foobar.Tag), "--concurrency", val).
					ExpectFailure().
					MatchErrKeyWords("Error:", "concurrency value should be at least 1").
					Exec()
			}
		})
	})
})

var _ = Describe("ORAS users:", func() {
	When("backing up a single tag", func() {
		It("should successfully backup an image to a directory", func() {
			tmpDir := GinkgoT().TempDir()
			outDir := filepath.Join(tmpDir, "backup-single-tag")
			srcRef := RegistryRef(ZOTHost, ImageRepo, foobar.Tag)
			foobarStates := append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageConfigStateKey(oras.MediaTypeUnknownConfig))

			ORAS("backup", "--output", outDir, srcRef).
				MatchStatus(foobarStates, true, len(foobarStates)).
				MatchKeyWords("Successfully backed up 1 tag(s)").
				Exec()

			// Verify backup output structure
			verifyBackupDirectoryStructure(outDir)

			// Verify backed up content
			compareBackupRef(srcRef, outDir, srcRef)
		})

		It("should backup an artifact with its referrers to a directory", func() {
			tmpDir := GinkgoT().TempDir()
			outDir := filepath.Join(tmpDir, "backup-single-tag-referrers")
			srcRef := RegistryRef(ZOTHost, ArtifactRepo, foobar.Tag)
			foobarStates := append(append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageReferrerConfigStateKeys[0]), foobar.ImageReferrersStateKeys...)

			ORAS("backup", "--output", outDir, Flags.IncludeReferrers, srcRef).
				MatchStatus(foobarStates, true, len(foobarStates)).
				MatchKeyWords("2 referrer(s)").
				MatchKeyWords("Successfully backed up 1 tag(s)").
				Exec()

			// Verify backup output structure
			verifyBackupDirectoryStructure(outDir)

			// Verify backed up content
			compareBackupRef(srcRef, outDir, srcRef)
			for _, referrerDgst := range backupReferrers(outDir, srcRef) {
				compareBackupRef(RegistryRef(ZOTHost, ArtifactRepo, referrerDgst), outDir, referrerDgst)
			}
		})

		It("should successfully backup a multi-arch artifact to a directory", func() {
			tmpDir := GinkgoT().TempDir()
			outDir := filepath.Join(tmpDir, "backup-multi-arch")
			srcRef := RegistryRef(ZOTHost, ArtifactRepo, ma.Tag)
			stateKeys := ma.IndexStateKeys

			ORAS("backup", "--output", outDir, srcRef).
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("Successfully backed up 1 tag(s)").
				Exec()

			// Verify backup output structure
			verifyBackupDirectoryStructure(outDir)

			// Verify backed up content
			compareBackupRef(srcRef, outDir, srcRef)
		})

		It("should successfully backup a multi-arch artifact with referrers to a directory", func() {
			tmpDir := GinkgoT().TempDir()
			outDir := filepath.Join(tmpDir, "backup-multi-arch-referrers")
			srcRef := RegistryRef(ZOTHost, ArtifactRepo, ma.Tag)
			stateKeys := append(ma.IndexStateKeys, ma.IndexZOTReferrerStateKey, ma.LinuxAMD64ReferrerConfigStateKey)

			ORAS("backup", "--output", outDir, Flags.IncludeReferrers, srcRef).
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("3 referrer(s)").
				MatchKeyWords("Successfully backed up 1 tag(s)").
				Exec()

			// Verify backup output structure
			verifyBackupDirectoryStructure(outDir)

			// Verify backed up content
			compareBackupRef(srcRef, outDir, srcRef)
			for _, referrerDgst := range backupReferrers(outDir, srcRef) {
				compareBackupRef(RegistryRef(ZOTHost, ArtifactRepo, referrerDgst), outDir, referrerDgst)
			}
		})

		It("should back up a multi-arch image (itself has no referrers), child images and referrers of the child images, to a directory", func() {
			tag := "v1.3.8"
			srcRef := RegistryRef(ZOTHost, ArtifactRepo, tag)
			tmpDir := GinkgoT().TempDir()
			outDir := filepath.Join(tmpDir, "backup-index-without-referrers")
			stateKeys := []match.StateKey{
				{Digest: "44136fa355b3", Name: "application/vnd.oci.empty.v1+json"},
				{Digest: "01fa0c3558d5", Name: "arm64"},
				{Digest: "2960eae76dd7", Name: "amd64"},
				{Digest: "ab01d6e284e8", Name: "application/vnd.oci.image.manifest.v1+json"},
				{Digest: "6aa11331ce0c", Name: "application/vnd.oci.image.manifest.v1+json"},
				{Digest: "58e0d01dbd27", Name: "signature"},
				{Digest: "ecbd32686867", Name: "referrerimage"},
				{Digest: "02746a135c9e", Name: "sbom"},
				{Digest: "553c18eccc8b", Name: "application/vnd.oci.image.index.v1+json"},
			}

			ORAS("backup", "--output", outDir, Flags.IncludeReferrers, srcRef).
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("3 referrer(s)").
				MatchKeyWords("Successfully backed up 1 tag(s)").
				Exec()

				// Verify backup output structure
			verifyBackupDirectoryStructure(outDir)

			// Verify backed up content
			// validate that the index is copied
			compareBackupRef(srcRef, outDir, srcRef)
			// validate that the child images are copied
			compareBackupRef(RegistryRef(ZOTHost, ArtifactRepo, "sha256:ab01d6e284e843d51fb5e753904a540f507a62361a5fd7e434e4f27b285ca5c9"), outDir, "sha256:ab01d6e284e843d51fb5e753904a540f507a62361a5fd7e434e4f27b285ca5c9")
			compareBackupRef(RegistryRef(ZOTHost, ArtifactRepo, "sha256:6aa11331ce0c766d6333b60dac98d584d98eea45fa93bbfc9b5bdb915ce3a43f"), outDir, "sha256:6aa11331ce0c766d6333b60dac98d584d98eea45fa93bbfc9b5bdb915ce3a43f")
			// validate that the referrers of the child images are copied
			compareBackupRef(RegistryRef(ZOTHost, ArtifactRepo, "sha256:359bac7f6a262e0f36e83b6b78ee3cc7a0bb8813e04d330328ca7ca9785e1e0b"), outDir, "sha256:359bac7f6a262e0f36e83b6b78ee3cc7a0bb8813e04d330328ca7ca9785e1e0b")
			compareBackupRef(RegistryRef(ZOTHost, ArtifactRepo, "sha256:938419ae89a9947476bbed93abc5eb7abf7d5708be69679fe6cc4b22afe8fdd5"), outDir, "sha256:938419ae89a9947476bbed93abc5eb7abf7d5708be69679fe6cc4b22afe8fdd5")
			compareBackupRef(RegistryRef(ZOTHost, ArtifactRepo, "sha256:20e7d3a6ce087c54238c18a3428853b50cdaf4478a9d00caa8304119b58ae8a9"), outDir, "sha256:20e7d3a6ce087c54238c18a3428853b50cdaf4478a9d00caa8304119b58ae8a9")
		})

		It("should successfully backup an image to tar file", func() {
			tmpDir := GinkgoT().TempDir()
			outTar := filepath.Join(tmpDir, "backup-single-tag.tar")
			srcRef := RegistryRef(ZOTHost, ImageRepo, foobar.Tag)
			foobarStates := append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageConfigStateKey(oras.MediaTypeUnknownConfig))

			ORAS("backup", "--output", outTar, srcRef).
				MatchStatus(foobarStates, true, len(foobarStates)).
				Exec()

			// Verify backup structure
			verifyBackupTarStructure(outTar)

			// Verify backed up content
			compareBackupRef(srcRef, outTar, srcRef)
		})

		It("should successfully backup an image to tar file (case insensitive)", func() {
			tmpDir := GinkgoT().TempDir()
			outTar := filepath.Join(tmpDir, "backup-single-tag.tAr")
			srcRef := RegistryRef(ZOTHost, ImageRepo, foobar.Tag)
			foobarStates := append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageConfigStateKey(oras.MediaTypeUnknownConfig))

			ORAS("backup", "--output", outTar, srcRef).
				MatchStatus(foobarStates, true, len(foobarStates)).
				Exec()

			// Verify backup structure
			verifyBackupTarStructure(outTar)

			// Verify backed up content
			compareBackupRef(srcRef, outTar, srcRef)
		})
	})

	When("backing up multiple tags", func() {
		It("should backup all specified tags", func() {
			tmpDir := GinkgoT().TempDir()
			outDir := filepath.Join(tmpDir, "backup-multiple-tags")
			srcTags := []string{foobar.Tag, ma.Tag}
			srcRefs := fmt.Sprintf("%s/%s:%s", ZOTHost, ArtifactRepo, strings.Join(srcTags, ","))
			// foobar state keys
			stateKeys := append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey)
			// ma state keys
			stateKeys = append(stateKeys, ma.IndexStateKeys...)

			// Specify multiple tags in format: repo:tag1,tag2
			ORAS("backup", "--output", outDir, srcRefs).
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("Successfully backed up 2 tag(s)").
				Exec()

			// Verify backup output structure
			verifyBackupDirectoryStructure(outDir)

			// Verify backed up content
			for _, tag := range srcTags {
				srcRef := RegistryRef(ZOTHost, ArtifactRepo, tag)
				compareBackupRef(srcRef, outDir, srcRef)
			}
		})

		It("should backup all specified tags with referrers", func() {
			tmpDir := GinkgoT().TempDir()
			outDir := filepath.Join(tmpDir, "backup-multiple-tags-referrers")
			srcTags := []string{foobar.Tag, ma.Tag}
			srcRefs := fmt.Sprintf("%s/%s:%s", ZOTHost, ArtifactRepo, strings.Join(srcTags, ","))
			// foobar state keys
			foobarStateKeys := append(append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageReferrerConfigStateKeys[0]), foobar.ImageReferrersStateKeys...)
			// ma state keys
			maStateKeys := append(ma.IndexStateKeys, ma.IndexZOTReferrerStateKey, ma.LinuxAMD64ReferrerConfigStateKey)
			// combined state keys
			stateKeys := append(foobarStateKeys, maStateKeys...)

			ORAS("backup", "--output", outDir, Flags.IncludeReferrers, srcRefs).
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("Successfully backed up 2 tag(s)").
				Exec()

			// Verify backup output structure
			verifyBackupDirectoryStructure(outDir)

			// Verify backed up content
			for _, tag := range srcTags {
				srcRef := RegistryRef(ZOTHost, ArtifactRepo, tag)
				compareBackupRef(srcRef, outDir, srcRef)
				for _, referrerDgst := range backupReferrers(outDir, srcRef) {
					compareBackupRef(RegistryRef(ZOTHost, ArtifactRepo, referrerDgst), outDir, referrerDgst)
				}
			}
		})

		It("should auto-discover and back up all tags when no tag is specified", func() {
			tmpDir := GinkgoT().TempDir()
			outDir := filepath.Join(tmpDir, "backup-discovered-tags")
			srcTags := []string{foobar.Tag, ma.Tag}
			// foobar state keys
			stateKeys := append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey)
			// ma state keys
			stateKeys = append(stateKeys, ma.IndexStateKeys...)

			// prepare test repo
			testRepo := backupTestRepo("backup-discovered-tags")
			for _, tag := range srcTags {
				prepare(RegistryRef(ZOTHost, ArtifactRepo, tag), RegistryRef(ZOTHost, testRepo, tag))
			}

			// test
			srcRefs := fmt.Sprintf("%s/%s", ZOTHost, testRepo)
			ORAS("backup", "--output", outDir, srcRefs).
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("Successfully backed up 2 tag(s)").
				Exec()

			// Verify backup output structure
			verifyBackupDirectoryStructure(outDir)

			// Verify backed up content
			for _, tag := range srcTags {
				compareBackupRef(RegistryRef(ZOTHost, ArtifactRepo, tag), outDir, RegistryRef(ZOTHost, testRepo, tag))
			}
		})

		It("should auto-discover and backup all tags with referrers when no tag is specified", func() {
			tmpDir := GinkgoT().TempDir()
			outDir := filepath.Join(tmpDir, "backup-discovered-tags-referrers")
			srcTags := []string{foobar.Tag, ma.Tag}
			// foobar state keys
			foobarStateKeys := append(append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageReferrerConfigStateKeys[0]), foobar.ImageReferrersStateKeys...)
			// ma state keys
			maStateKeys := append(ma.IndexStateKeys, ma.IndexZOTReferrerStateKey, ma.LinuxAMD64ReferrerConfigStateKey)
			// combined state keys
			stateKeys := append(foobarStateKeys, maStateKeys...)

			// prepare test repo
			testRepo := backupTestRepo("backup-discovered-tags-referrers")
			for _, tag := range srcTags {
				ORAS("cp", RegistryRef(ZOTHost, ArtifactRepo, tag), RegistryRef(ZOTHost, testRepo, tag), "-r").
					WithDescription("copying tag to test repo").
					Exec()
			}

			srcRefs := fmt.Sprintf("%s/%s", ZOTHost, testRepo)
			ORAS("backup", "--output", outDir, Flags.IncludeReferrers, srcRefs).
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("Successfully backed up 2 tag(s)").
				Exec()

			// Verify backup output structure
			verifyBackupDirectoryStructure(outDir)

			// Verify backed up content
			for _, tag := range srcTags {
				srcRef := RegistryRef(ZOTHost, ArtifactRepo, tag)
				layoutRef := RegistryRef(ZOTHost, testRepo, tag)
				compareBackupRef(srcRef, outDir, layoutRef)
				for _, referrerDgst := range backupReferrers(outDir, layoutRef) {
					compareBackupRef(RegistryRef(ZOTHost, ArtifactRepo, referrerDgst), outDir, referrerDgst)
				}
			}
		})
	})

	When("backing up with --platform", func() {
		maArm64Digest := ma.LinuxARM64.Digest.String()

		It("should back up only the matching manifest of a multi-arch artifact", func() {
			outDir := filepath.Join(GinkgoT().TempDir(), "backup-platform")
			srcRef := RegistryRef(ZOTHost, ArtifactRepo, ma.Tag)

			ORAS("backup", "--output", outDir, "--platform", "linux/amd64", srcRef).
				MatchKeyWords("Successfully backed up 1 tag(s)").
				Exec()

			verifyBackupDirectoryStructure(outDir)
			// the stored tag points at the linux/amd64 manifest, not the index
			stored := ORAS("manifest", "fetch", ociLayoutPath, outDir, srcRef).Exec().Out.Contents()
			Expect(string(stored)).To(Equal(ma.LinuxAMD64Manifest))
			ORAS("manifest", "fetch", ociLayoutPath, outDir, ma.Digest).ExpectFailure().Exec()
		})

		It("should back up the referrers of the matching manifest only", func() {
			outDir := filepath.Join(GinkgoT().TempDir(), "backup-platform-referrers")
			srcRef := RegistryRef(ZOTHost, ArtifactRepo, ma.Tag)

			ORAS("backup", "--output", outDir, "--platform", "linux/amd64", Flags.IncludeReferrers, srcRef).
				MatchKeyWords("Successfully backed up 1 tag(s)").
				Exec()

			verifyBackupDirectoryStructure(outDir)
			compareBackupRef(RegistryRef(ZOTHost, ArtifactRepo, ma.LinuxAMD64Referrer.Digest.String()), outDir, ma.LinuxAMD64Referrer.Digest.String())
			// referrers of the index are not kept
			ORAS("manifest", "fetch", ociLayoutPath, outDir, ma.IndexReferrerDigest).ExpectFailure().Exec()
		})

		It("should fail when a named tag has no manifest for the platform", func() {
			outDir := filepath.Join(GinkgoT().TempDir(), "backup-platform-mismatch")
			srcRef := RegistryRef(ZOTHost, ArtifactRepo, ma.Tag)

			ORAS("backup", "--output", outDir, "--platform", "linux/s390x", srcRef).
				ExpectFailure().
				MatchErrKeyWords("no manifest for the requested platform").
				Exec()
		})

		It("should back up an artifact without a platform in full", func() {
			outDir := filepath.Join(GinkgoT().TempDir(), "backup-platform-artifact")
			srcRef := RegistryRef(ZOTHost, ArtifactRepo, foobar.Tag)

			ORAS("backup", "--output", outDir, "--platform", "linux/amd64", srcRef).
				MatchKeyWords("Successfully backed up 1 tag(s)").
				Exec()

			verifyBackupDirectoryStructure(outDir)
			compareBackupRef(srcRef, outDir, srcRef)
		})

		It("should back up a whole repository with images and artifacts", func() {
			testRepo := backupTestRepo("backup-platform-repo")
			for _, tag := range []string{foobar.Tag, ma.Tag} {
				prepare(RegistryRef(ZOTHost, ArtifactRepo, tag), RegistryRef(ZOTHost, testRepo, tag))
			}
			srcRepo := fmt.Sprintf("%s/%s", ZOTHost, testRepo)

			By("keeping the matching manifest of the image and the artifact in full")
			outDir := filepath.Join(GinkgoT().TempDir(), "backup-platform-repo")
			ORAS("backup", "--output", outDir, "--platform", "linux/arm64", srcRepo).
				MatchKeyWords("Successfully backed up 2 tag(s)").
				Exec()
			compareBackupRef(RegistryRef(ZOTHost, ArtifactRepo, foobar.Tag), outDir, RegistryRef(ZOTHost, testRepo, foobar.Tag))
			compareBackupRef(RegistryRef(ZOTHost, ArtifactRepo, maArm64Digest), outDir, RegistryRef(ZOTHost, testRepo, ma.Tag))

			By("skipping the image when it has no manifest for the platform")
			outDir = filepath.Join(GinkgoT().TempDir(), "backup-platform-repo-skip")
			ORAS("backup", "--output", outDir, "--platform", "linux/s390x", srcRepo).
				MatchKeyWords("Successfully backed up 1 tag(s)").
				MatchErrKeyWords("skipping tag", ma.Tag).
				Exec()
			compareBackupRef(RegistryRef(ZOTHost, ArtifactRepo, foobar.Tag), outDir, RegistryRef(ZOTHost, testRepo, foobar.Tag))
		})

		It("should restore what it backed up", func() {
			backupDir := filepath.Join(GinkgoT().TempDir(), "backup-platform-round-trip")
			ORAS("backup", "--output", backupDir, "--platform", "linux/amd64", RegistryRef(ZOTHost, ArtifactRepo, ma.Tag)).Exec()

			testRepo := backupTestRepo("restore-platform")
			ORAS("restore", "--input", backupDir, Flags.ExcludeReferrers, fmt.Sprintf("%s/%s", ZOTHost, testRepo)).Exec()

			// the backup records the source repository, which is restored under the target
			restored := ORAS("manifest", "fetch", RegistryRef(ZOTHost, testRepo+"/"+ArtifactRepo, ma.Tag)).Exec().Out.Contents()
			Expect(string(restored)).To(Equal(ma.LinuxAMD64Manifest))
		})

		It("should warn when restore replaces a multi-arch tag with a platform manifest", func() {
			backupDir := filepath.Join(GinkgoT().TempDir(), "backup-platform-warning")
			srcRef := RegistryRef(ZOTHost, ArtifactRepo, ma.Tag)
			ORAS("backup", "--output", backupDir, "--platform", "linux/amd64", srcRef).Exec()

			testRepo := backupTestRepo("restore-platform-warning")
			restoredRef := RegistryRef(ZOTHost, testRepo+"/"+ArtifactRepo, ma.Tag)
			prepare(srcRef, restoredRef)

			ORAS("restore", "--input", backupDir, Flags.ExcludeReferrers, fmt.Sprintf("%s/%s", ZOTHost, testRepo)).
				MatchErrKeyWords("currently points to multi-platform index", ma.Digest, "restoring replaces it with a single-platform manifest").
				Exec()

			restored := ORAS("manifest", "fetch", restoredRef).Exec().Out.Contents()
			Expect(string(restored)).To(Equal(ma.LinuxAMD64Manifest))
		})
	})

	When("using --distribution-spec flag", func() {
		It("should backup using referrers API with --distribution-spec v1.1-referrers-api", func() {
			tmpDir := GinkgoT().TempDir()
			outDir := filepath.Join(tmpDir, "backup-referrers-api")
			srcRef := RegistryRef(ZOTHost, ArtifactRepo, foobar.Tag)

			ORAS("backup", "--output", outDir, Flags.IncludeReferrers, Flags.DistributionSpec, "v1.1-referrers-api", srcRef).
				MatchKeyWords("2 referrer(s)").
				Exec()

			// Verify backup output structure
			verifyBackupDirectoryStructure(outDir)

			// Verify backed up content
			compareBackupRef(srcRef, outDir, srcRef)
			for _, referrerDgst := range backupReferrers(outDir, srcRef) {
				compareBackupRef(RegistryRef(ZOTHost, ArtifactRepo, referrerDgst), outDir, referrerDgst)
			}
		})

		It("should backup using tag schema with --distribution-spec v1.1-referrers-tag", func() {
			tmpDir := GinkgoT().TempDir()
			outDir := filepath.Join(tmpDir, "backup-referrers-tag")
			srcRef := RegistryRef(FallbackHost, ArtifactRepo, foobar.Tag)

			ORAS("backup", "--output", outDir, Flags.IncludeReferrers, Flags.DistributionSpec, "v1.1-referrers-tag", srcRef).
				MatchKeyWords("2 referrer(s)").
				Exec()

			// Verify backup output structure
			verifyBackupDirectoryStructure(outDir)

			// Verify backed up content
			compareBackupRef(srcRef, outDir, srcRef)
			for _, referrerDgst := range backupReferrers(outDir, srcRef) {
				compareBackupRef(RegistryRef(ZOTHost, ArtifactRepo, referrerDgst), outDir, referrerDgst)
			}
		})
	})

	When("using --concurrency flag", func() {
		It("should successfully backup with custom concurrency level", func() {
			tmpDir := GinkgoT().TempDir()
			outDir := filepath.Join(tmpDir, "concurrency")
			src := RegistryRef(ZOTHost, ImageRepo, ma.Tag)

			stateKeys := ma.IndexStateKeys

			ORAS("backup", "--output", outDir, "--concurrency", "1", src).
				MatchStatus(stateKeys, true, len(stateKeys)).
				Exec()

			// Verify directory structure was created correctly
			verifyBackupDirectoryStructure(outDir)
			// Verify backed up content
			compareBackupRef(src, outDir, src)
		})
	})

	When("verifying the reference annotation", func() {
		It("should store the full reference in org.opencontainers.image.ref.name", func() {
			tmpDir := GinkgoT().TempDir()
			outDir := filepath.Join(tmpDir, "backup-full-reference")
			repo := backupTestRepo("full-reference")
			tag := "v1.0"
			srcRef := RegistryRef(ZOTHost, repo, tag)

			prepare(RegistryRef(ZOTHost, ArtifactRepo, foobar.Tag), srcRef)

			ORAS("backup", "--output", outDir, srcRef).Exec()

			verifyBackupDirectoryStructure(outDir)

			// Read and verify index.json annotations
			indexPath := filepath.Join(outDir, "index.json")
			indexBytes, err := os.ReadFile(indexPath)
			Expect(err).ToNot(HaveOccurred())

			var index ocispec.Index
			err = json.Unmarshal(indexBytes, &index)
			Expect(err).ToNot(HaveOccurred())
			Expect(index.Manifests).ToNot(BeEmpty())

			// Verify the full reference is stored in ref.name and no title is set
			annotations := index.Manifests[0].Annotations
			expectedFullRef := fmt.Sprintf("%s/%s:%s", ZOTHost, repo, tag)
			Expect(annotations[ocispec.AnnotationRefName]).To(Equal(expectedFullRef))
			Expect(annotations).ToNot(HaveKey(ocispec.AnnotationTitle))
		})
	})

	When("handling error cases", func() {
		It("should fail when output directory cannot be created", func() {
			// Create a file that will conflict with our output path
			tmpDir := GinkgoT().TempDir()
			conflictFile := filepath.Join(tmpDir, "backup-conflict-file")
			defer func() {
				_ = os.RemoveAll(conflictFile)
			}()

			// Create a file (not a directory)
			fp, err := os.Create(conflictFile)
			Expect(err).ToNot(HaveOccurred())
			_ = fp.Close()

			// Try to use the file as an output directory
			ORAS("backup", "--output", conflictFile, RegistryRef(ZOTHost, ArtifactRepo, foobar.Tag)).
				ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})

		It("should fail with appropriate error when output path is a directory but tar file expected", func() {
			// Create a directory that will conflict with the tar output path
			tmpDir := GinkgoT().TempDir()
			tarDirPath := filepath.Join(tmpDir, "backup-dir-conflict.tar")

			// Create a directory with the .tar extension
			err := os.MkdirAll(tarDirPath, 0755)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				_ = os.RemoveAll(tarDirPath)
			}()

			// Try to use the directory as a tar output file
			ORAS("backup", "--output", tarDirPath, RegistryRef(ZOTHost, ImageRepo, foobar.Tag)).
				ExpectFailure().
				MatchErrKeyWords("Error:", "already exists and is a directory").
				MatchErrKeyWords("To back up to a tar archive, please specify a different output file name or remove the existing directory.").
				Exec()
		})

		It("should fail when the repository doesn't exist", func() {
			tmpDir := GinkgoT().TempDir()
			outDir := filepath.Join(tmpDir, "backup-nonexistent-repo")

			// Use a repository name that definitely doesn't exist
			nonexistentRepo := fmt.Sprintf("nonexistent-repo-%d-%d", GinkgoRandomSeed(), time.Now().UnixNano())
			srcRef := fmt.Sprintf("%s/%s", ZOTHost, nonexistentRepo)

			ORAS("backup", "--output", outDir, srcRef).ExpectFailure().
				MatchErrKeyWords("Error response from registry:").Exec()
		})

		It("should fail when no tags are found in the repository", func() {
			tmpDir := GinkgoT().TempDir()
			outDir := filepath.Join(tmpDir, "backup-no-tags")

			// Setup a test repository with no tags
			testRepo := backupTestRepo("backup-repo-no-tags")
			srcRef := fmt.Sprintf("%s/%s", ZOTHost, testRepo)
			prepare(RegistryRef(ZOTHost, ArtifactRepo, foobar.Digest), RegistryRef(ZOTHost, testRepo, foobar.Digest))

			ORAS("backup", "--output", outDir, srcRef).ExpectFailure().
				MatchErrKeyWords("Error:", "no tags found").
				MatchErrKeyWords("oras repo tags"). // test recommendation
				Exec()
		})

		It("should fail when a specified tag doesn't exist", func() {
			tmpDir := GinkgoT().TempDir()
			outDir := filepath.Join(tmpDir, "backup-nonexistent-tag")

			// Try to backup a nonexistent tag from this repo
			srcRef := RegistryRef(ZOTHost, ImageRepo, InvalidTag)

			ORAS("backup", "--output", outDir, srcRef).ExpectFailure().
				MatchErrKeyWords("Error:", InvalidTag, "not found").Exec()
		})
	})
})
