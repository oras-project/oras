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
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/oras-project/oras-go/v3"
	"oras.land/oras/test/e2e/internal/testdata/feature"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	ma "oras.land/oras/test/e2e/internal/testdata/multi_arch"
	. "oras.land/oras/test/e2e/internal/utils"
	"oras.land/oras/test/e2e/internal/utils/match"
)

func restoreTestRepo(text string) string {
	return fmt.Sprintf("command/restore/%d/%s", GinkgoRandomSeed(), text)
}

var _ = Describe("ORAS beginners:", func() {
	When("running restore command", func() {
		It("should show help description with experimental flag", func() {
			out := ORAS("restore", "--help").MatchKeyWords(ExampleDesc).Exec().Out
			gomega.Expect(out).Should(gbytes.Say(regexp.QuoteMeta(feature.Experimental.Mark)))
		})

		It("should fail when no reference provided", func() {
			ORAS("restore").ExpectFailure().MatchErrKeyWords("Error:").Exec()
		})

		It("should fail when no input path provided", func() {
			ORAS("restore", fmt.Sprintf("%s/%s", ZOTHost, ImageRepo)).ExpectFailure().
				MatchErrKeyWords("Error:", `required flag(s) "input" not set`).Exec()
		})

		It("should fail when the input path is empty", func() {
			ORAS("restore", "--input", "", fmt.Sprintf("%s/%s", ZOTHost, ImageRepo)).ExpectFailure().
				MatchErrKeyWords("Error:", `the input path cannot be empty`).Exec()
		})

		It("should fail when input doesn't exist", func() {
			nonExistentPath := filepath.Join(GinkgoT().TempDir(), "non-existent-path")
			ORAS("restore", "--input", nonExistentPath, fmt.Sprintf("%s/%s", ZOTHost, ImageRepo)).ExpectFailure().
				MatchErrKeyWords("Error:").Exec()
		})

		It("should fail with appropriate error when digest is provided", func() {
			tmpDir := GinkgoT().TempDir()
			ORAS("restore", "--input", tmpDir, "invalid/format@sha256:digest").ExpectFailure().
				MatchErrKeyWords("Error:", "a tag or digest cannot be specified in the restore target").Exec()
		})

		It("should fail with appropriate error when invalid tag format provided", func() {
			tmpDir := GinkgoT().TempDir()
			ORAS("restore", "--input", tmpDir, "localhost:5000/repo:invalid+tag").ExpectFailure().
				MatchErrKeyWords("Error:").Exec()
		})

		It("should fail with appropriate error when tag is provided with digest", func() {
			tmpDir := GinkgoT().TempDir()
			ORAS("restore", "--input", tmpDir, "invalid/format:v1,@sha256:123abc").ExpectFailure().
				MatchErrKeyWords("Error:", "a tag or digest cannot be specified in the restore target").Exec()
		})

		It("should fail if given an invalid value for concurrency", func() {
			tmpDir := GinkgoT().TempDir()
			for _, val := range []string{"0", "-1"} {
				ORAS("restore", "--input", tmpDir, fmt.Sprintf("%s/%s", ZOTHost, ImageRepo), "--concurrency", val).
					ExpectFailure().
					MatchErrKeyWords("Error:", "concurrency value should be at least 1").
					Exec()
			}
		})
	})
})

var _ = Describe("ORAS users:", func() {
	When("restoring a single tag", func() {
		It("should successfully restore an image without referrers from a directory", func() {
			// Prepare a backup to restore from
			tmpDir := GinkgoT().TempDir()
			srcRef := RegistryRef(ZOTHost, ImageRepo, foobar.Tag)
			backupDir := filepath.Join(tmpDir, "backup-single-tag")

			ORAS("backup", "--output", backupDir, srcRef).Exec()

			// Prepare target repo for restore
			testRepo := restoreTestRepo("restore-single-tag-dir")
			// the backup records the source repository, which is restored under the target
			restoredRepo := testRepo + "/" + ImageRepo
			dstRef := fmt.Sprintf("%s/%s", ZOTHost, testRepo)

			// Restore from backup directory
			foobarStates := append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageConfigStateKey(oras.MediaTypeUnknownConfig))
			ORAS("restore", "--input", backupDir, Flags.ExcludeReferrers, dstRef).
				MatchStatus(foobarStates, true, len(foobarStates)).
				MatchKeyWords("0 referrer(s)").
				MatchKeyWords("Successfully restored 1 tag(s)").
				Exec()

			// Verify restored content
			CompareRef(srcRef, RegistryRef(ZOTHost, restoredRepo, foobar.Tag))
		})

		It("should restore from a tag-only backup created by an earlier version", func() {
			// Earlier versions of oras backup stored only the tag in
			// org.opencontainers.image.ref.name, as oras cp does for an OCI layout.
			tmpDir := GinkgoT().TempDir()
			srcRef := RegistryRef(ZOTHost, ImageRepo, foobar.Tag)
			backupDir := filepath.Join(tmpDir, "backup-tag-only")
			ORAS("cp", srcRef, Flags.ToLayout, LayoutRef(backupDir, foobar.Tag)).Exec()

			testRepo := restoreTestRepo("restore-tag-only")
			dstRef := RegistryRef(ZOTHost, testRepo, foobar.Tag)
			ORAS("restore", "--input", backupDir, Flags.ExcludeReferrers, fmt.Sprintf("%s/%s", ZOTHost, testRepo)).
				MatchKeyWords("Successfully restored 1 tag(s)").
				Exec()

			CompareRef(srcRef, dstRef)
		})

		It("should restore an artifact with its referrers from a directory", func() {
			// Create a backup with referrers
			tmpDir := GinkgoT().TempDir()
			backupDir := filepath.Join(tmpDir, "backup-with-referrers")
			srcRef := RegistryRef(ZOTHost, ArtifactRepo, foobar.Tag)

			ORAS("backup", "--output", backupDir, Flags.IncludeReferrers, srcRef).Exec()

			// Create target repo for restore
			testRepo := restoreTestRepo("restore-with-referrers")
			// the backup records the source repository, which is restored under the target
			restoredRepo := testRepo + "/" + ArtifactRepo
			dstRef := fmt.Sprintf("%s/%s", ZOTHost, testRepo)

			// Restore from backup with referrers
			foobarStates := append(append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageReferrerConfigStateKeys[0]), foobar.ImageReferrersStateKeys...)

			ORAS("restore", "--input", backupDir, dstRef).
				MatchStatus(foobarStates, true, len(foobarStates)).
				MatchKeyWords("2 referrer(s)").
				MatchKeyWords("Successfully restored 1 tag(s)").
				Exec()

			// Verify restored content
			CompareRef(srcRef, RegistryRef(ZOTHost, restoredRepo, foobar.Tag))

			// Verify referrers were restored
			referrers := ORAS("discover", RegistryRef(ZOTHost, restoredRepo, foobar.Tag), "--format", "go-template={{range .referrers}}{{println .digest}}{{end}}").Exec().Out.Contents()
			for referrerDgst := range strings.SplitSeq(strings.TrimSpace(string(referrers)), "\n") {
				CompareRef(RegistryRef(ZOTHost, ArtifactRepo, referrerDgst), RegistryRef(ZOTHost, restoredRepo, referrerDgst))
			}
		})

		It("should successfully restore a multi-arch artifact without referrers from a directory", func() {
			// Create a backup to restore from
			tmpDir := GinkgoT().TempDir()
			backupDir := filepath.Join(tmpDir, "backup-multi-arch")
			srcRef := RegistryRef(ZOTHost, ArtifactRepo, ma.Tag)

			ORAS("backup", "--output", backupDir, srcRef).Exec()

			// Create target repo for restore
			testRepo := restoreTestRepo("restore-multi-arch")
			// the backup records the source repository, which is restored under the target
			restoredRepo := testRepo + "/" + ArtifactRepo
			dstRef := fmt.Sprintf("%s/%s", ZOTHost, testRepo)

			// Restore from backup directory
			stateKeys := ma.IndexStateKeys

			ORAS("restore", "--input", backupDir, Flags.ExcludeReferrers, dstRef).
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("0 referrer(s)").
				MatchKeyWords("Successfully restored 1 tag(s)").
				Exec()

			// Verify restored content
			CompareRef(srcRef, RegistryRef(ZOTHost, restoredRepo, ma.Tag))
		})

		It("should successfully restore a multi-arch artifact with referrers from a directory", func() {
			// Create a backup with multi-arch and referrers
			tmpDir := GinkgoT().TempDir()
			backupDir := filepath.Join(tmpDir, "backup-multi-arch-referrers")
			srcRef := RegistryRef(ZOTHost, ArtifactRepo, ma.Tag)

			ORAS("backup", "--output", backupDir, Flags.IncludeReferrers, srcRef).Exec()

			// Create target repo for restore
			testRepo := restoreTestRepo("restore-multi-arch-referrers")
			// the backup records the source repository, which is restored under the target
			restoredRepo := testRepo + "/" + ArtifactRepo
			dstRef := fmt.Sprintf("%s/%s", ZOTHost, testRepo)

			// Restore multi-arch with referrers
			stateKeys := append(ma.IndexStateKeys, ma.IndexZOTReferrerStateKey, ma.LinuxAMD64ReferrerConfigStateKey)

			ORAS("restore", "--input", backupDir, dstRef).
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("3 referrer(s)").
				MatchKeyWords("Successfully restored 1 tag(s)").
				Exec()

			// Verify restored content
			CompareRef(srcRef, RegistryRef(ZOTHost, restoredRepo, ma.Tag))

			// Verify referrers were restored
			referrers := ORAS("discover", RegistryRef(ZOTHost, restoredRepo, ma.Tag), "--format", "go-template={{range .referrers}}{{println .digest}}{{end}}").Exec().Out.Contents()
			for referrerDgst := range strings.SplitSeq(strings.TrimSpace(string(referrers)), "\n") {
				CompareRef(RegistryRef(ZOTHost, ArtifactRepo, referrerDgst), RegistryRef(ZOTHost, restoredRepo, referrerDgst))
			}
		})

		It("should restore a multi-arch image with child images and referrers of the child images", func() {
			// Create a backup with a multi-arch image that has no direct referrers
			// but has child images with referrers
			tmpDir := GinkgoT().TempDir()
			backupDir := filepath.Join(tmpDir, "backup-multi-arch-child-referrers")
			tag := "v1.3.8"
			srcRef := RegistryRef(ZOTHost, ArtifactRepo, tag)

			ORAS("backup", "--output", backupDir, Flags.IncludeReferrers, srcRef).Exec()

			// Create target repo for restore
			testRepo := restoreTestRepo("restore-multi-arch-child-referrers")
			// the backup records the source repository, which is restored under the target
			restoredRepo := testRepo + "/" + ArtifactRepo
			dstRef := fmt.Sprintf("%s/%s", ZOTHost, testRepo)

			// Restore from backup directory
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

			ORAS("restore", "--input", backupDir, dstRef).
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("3 referrer(s)").
				MatchKeyWords("Successfully restored 1 tag(s)").
				Exec()

			// Verify restored content
			// validate that the index is restored correctly
			CompareRef(srcRef, RegistryRef(ZOTHost, restoredRepo, tag))

			// validate that the child images are restored correctly
			CompareRef(
				RegistryRef(ZOTHost, ArtifactRepo, "sha256:ab01d6e284e843d51fb5e753904a540f507a62361a5fd7e434e4f27b285ca5c9"),
				RegistryRef(ZOTHost, restoredRepo, "sha256:ab01d6e284e843d51fb5e753904a540f507a62361a5fd7e434e4f27b285ca5c9"),
			)
			CompareRef(
				RegistryRef(ZOTHost, ArtifactRepo, "sha256:6aa11331ce0c766d6333b60dac98d584d98eea45fa93bbfc9b5bdb915ce3a43f"),
				RegistryRef(ZOTHost, restoredRepo, "sha256:6aa11331ce0c766d6333b60dac98d584d98eea45fa93bbfc9b5bdb915ce3a43f"),
			)

			// validate that the referrers of the child images are restored correctly
			CompareRef(
				RegistryRef(ZOTHost, ArtifactRepo, "sha256:359bac7f6a262e0f36e83b6b78ee3cc7a0bb8813e04d330328ca7ca9785e1e0b"),
				RegistryRef(ZOTHost, restoredRepo, "sha256:359bac7f6a262e0f36e83b6b78ee3cc7a0bb8813e04d330328ca7ca9785e1e0b"),
			)
			CompareRef(
				RegistryRef(ZOTHost, ArtifactRepo, "sha256:938419ae89a9947476bbed93abc5eb7abf7d5708be69679fe6cc4b22afe8fdd5"),
				RegistryRef(ZOTHost, restoredRepo, "sha256:938419ae89a9947476bbed93abc5eb7abf7d5708be69679fe6cc4b22afe8fdd5"),
			)
			CompareRef(
				RegistryRef(ZOTHost, ArtifactRepo, "sha256:20e7d3a6ce087c54238c18a3428853b50cdaf4478a9d00caa8304119b58ae8a9"),
				RegistryRef(ZOTHost, restoredRepo, "sha256:20e7d3a6ce087c54238c18a3428853b50cdaf4478a9d00caa8304119b58ae8a9"),
			)
		})

		It("should successfully restore an image without referrers from a tar file", func() {
			// Create a backup tar file
			tmpDir := GinkgoT().TempDir()
			backupTar := filepath.Join(tmpDir, "backup-single-tag.tar")
			srcRef := RegistryRef(ZOTHost, ImageRepo, foobar.Tag)

			ORAS("backup", "--output", backupTar, srcRef).Exec()

			// Create target repo for restore
			testRepo := restoreTestRepo("restore-single-tag-tar")
			// the backup records the source repository, which is restored under the target
			restoredRepo := testRepo + "/" + ImageRepo
			dstRef := fmt.Sprintf("%s/%s", ZOTHost, testRepo)

			// Restore from backup tar
			foobarStates := append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageConfigStateKey(oras.MediaTypeUnknownConfig))

			ORAS("restore", "--input", backupTar, Flags.ExcludeReferrers, dstRef).
				MatchStatus(foobarStates, true, len(foobarStates)).
				MatchKeyWords("0 referrer(s)").
				MatchKeyWords("Successfully restored 1 tag(s)").
				Exec()

			// Verify restored content
			CompareRef(srcRef, RegistryRef(ZOTHost, restoredRepo, foobar.Tag))
		})

		It("should restore an artifact with its referrers from a tar file", func() {
			// Create a backup tar with referrers
			tmpDir := GinkgoT().TempDir()
			backupTar := filepath.Join(tmpDir, "backup-with-referrers.tar")
			srcRef := RegistryRef(ZOTHost, ArtifactRepo, foobar.Tag)

			ORAS("backup", "--output", backupTar, Flags.IncludeReferrers, srcRef).Exec()

			// Create target repo for restore
			testRepo := restoreTestRepo("restore-with-referrers-tar")
			// the backup records the source repository, which is restored under the target
			restoredRepo := testRepo + "/" + ArtifactRepo
			dstRef := fmt.Sprintf("%s/%s", ZOTHost, testRepo)

			// Restore from backup tar with referrers
			foobarStates := append(append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageReferrerConfigStateKeys[0]), foobar.ImageReferrersStateKeys...)

			ORAS("restore", "--input", backupTar, dstRef).
				MatchStatus(foobarStates, true, len(foobarStates)).
				MatchKeyWords("2 referrer(s)").
				MatchKeyWords("Successfully restored 1 tag(s)").
				Exec()

			// Verify restored content
			CompareRef(srcRef, RegistryRef(ZOTHost, restoredRepo, foobar.Tag))

			// Verify referrers were restored
			referrers := ORAS("discover", RegistryRef(ZOTHost, restoredRepo, foobar.Tag), "--format", "go-template={{range .referrers}}{{println .digest}}{{end}}").Exec().Out.Contents()
			for referrerDgst := range strings.SplitSeq(strings.TrimSpace(string(referrers)), "\n") {
				CompareRef(RegistryRef(ZOTHost, ArtifactRepo, referrerDgst), RegistryRef(ZOTHost, restoredRepo, referrerDgst))
			}
		})

		It("should successfully restore a multi-arch artifact without referrers from a tar file", func() {
			// Create a backup tar
			tmpDir := GinkgoT().TempDir()
			backupTar := filepath.Join(tmpDir, "backup-multi-arch.tar")
			srcRef := RegistryRef(ZOTHost, ArtifactRepo, ma.Tag)

			ORAS("backup", "--output", backupTar, srcRef).Exec()

			// Create target repo for restore
			testRepo := restoreTestRepo("restore-multi-arch-tar")
			// the backup records the source repository, which is restored under the target
			restoredRepo := testRepo + "/" + ArtifactRepo
			dstRef := fmt.Sprintf("%s/%s", ZOTHost, testRepo)

			// Restore from backup tar
			stateKeys := ma.IndexStateKeys

			ORAS("restore", "--input", backupTar, Flags.ExcludeReferrers, dstRef).
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("0 referrer(s)").
				MatchKeyWords("Successfully restored 1 tag(s)").
				Exec()

			// Verify restored content
			CompareRef(srcRef, RegistryRef(ZOTHost, restoredRepo, ma.Tag))
		})
	})

	When("restoring multiple tags", func() {
		It("should restore all tags without referrers when no specific tags are provided", func() {
			// Create a backup with multiple tags
			tmpDir := GinkgoT().TempDir()
			backupDir := filepath.Join(tmpDir, "backup-all-tags")
			srcTags := []string{foobar.Tag, ma.Tag}

			// Prepare test repo with multiple tags for backup
			testSrcRepo := restoreTestRepo("backup-all-tags-src")
			for _, tag := range srcTags {
				prepare(RegistryRef(ZOTHost, ArtifactRepo, tag), RegistryRef(ZOTHost, testSrcRepo, tag))
			}

			// Backup all tags from source repo
			srcRepoRef := fmt.Sprintf("%s/%s", ZOTHost, testSrcRepo)
			ORAS("backup", "--output", backupDir, srcRepoRef).Exec()

			// Create target repo for restore
			testDstRepo := restoreTestRepo("restore-all-tags")
			// the backup records the source repository, which is restored under the target
			restoredRepo := testDstRepo + "/" + testSrcRepo
			dstRepoRef := fmt.Sprintf("%s/%s", ZOTHost, testDstRepo)

			// Restore all tags from backup
			stateKeys := append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey)
			stateKeys = append(stateKeys, ma.IndexStateKeys...)

			ORAS("restore", "--input", backupDir, Flags.ExcludeReferrers, dstRepoRef).
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("0 referrer(s)").
				MatchKeyWords("Successfully restored 2 tag(s)").
				Exec()

			// Verify all tags were restored
			for _, tag := range srcTags {
				CompareRef(
					RegistryRef(ZOTHost, testSrcRepo, tag),
					RegistryRef(ZOTHost, restoredRepo, tag),
				)
			}
		})

		It("should restore all tags with referrers when no specific tags are provided", func() {
			// Create a backup with multiple tags and their referrers
			tmpDir := GinkgoT().TempDir()
			backupDir := filepath.Join(tmpDir, "backup-all-tags-referrers")
			srcTags := []string{foobar.Tag, ma.Tag}

			// Prepare test repo with multiple tags for backup
			testSrcRepo := restoreTestRepo("backup-all-tags-referrers-src")
			for _, tag := range srcTags {
				ORAS("cp", RegistryRef(ZOTHost, ArtifactRepo, tag), RegistryRef(ZOTHost, testSrcRepo, tag), "-r").
					WithDescription("copying tag to test repo").
					Exec()
			}

			// Backup all tags and referrers from source repo
			srcRepoRef := fmt.Sprintf("%s/%s", ZOTHost, testSrcRepo)
			ORAS("backup", "--output", backupDir, Flags.IncludeReferrers, srcRepoRef).Exec()

			// Create target repo for restore
			testDstRepo := restoreTestRepo("restore-all-tags-referrers")
			// the backup records the source repository, which is restored under the target
			restoredRepo := testDstRepo + "/" + testSrcRepo
			dstRepoRef := fmt.Sprintf("%s/%s", ZOTHost, testDstRepo)

			// Restore all tags and their referrers from backup
			// foobar state keys with referrers
			foobarStateKeys := append(append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageReferrerConfigStateKeys[0]), foobar.ImageReferrersStateKeys...)
			// ma state keys with referrers
			maStateKeys := append(ma.IndexStateKeys, ma.IndexZOTReferrerStateKey, ma.LinuxAMD64ReferrerConfigStateKey)
			// combined state keys
			stateKeys := append(foobarStateKeys, maStateKeys...)

			ORAS("restore", "--input", backupDir, dstRepoRef).
				MatchStatus(stateKeys, true, len(stateKeys)).
				MatchKeyWords("Successfully restored 2 tag(s)").
				Exec()

			// Verify all tags and their referrers were restored
			for _, tag := range srcTags {
				srcRef := RegistryRef(ZOTHost, testSrcRepo, tag)
				dstRef := RegistryRef(ZOTHost, restoredRepo, tag)

				CompareRef(srcRef, dstRef)

				// Verify referrers were restored for each tag
				referrers := ORAS("discover", dstRef, "--format", "go-template={{range .referrers}}{{println .digest}}{{end}}").Exec().Out.Contents()
				for referrerDgst := range strings.SplitSeq(strings.TrimSpace(string(referrers)), "\n") {
					CompareRef(RegistryRef(ZOTHost, testSrcRepo, referrerDgst), RegistryRef(ZOTHost, restoredRepo, referrerDgst))
				}
			}
		})
	})

	When("using --dry-run", func() {
		It("should simulate restore without referrers", func() {
			// Create a backup
			tmpDir := GinkgoT().TempDir()
			backupDir := filepath.Join(tmpDir, "backup-dry-run")
			srcRef := RegistryRef(ZOTHost, ImageRepo, foobar.Tag)

			ORAS("backup", "--output", backupDir, srcRef).Exec()

			// Create target repo for restore
			testRepo := restoreTestRepo("restore-dry-run")
			dstRef := fmt.Sprintf("%s/%s", ZOTHost, testRepo)
			restoredRef := RegistryRef(ZOTHost, testRepo+"/"+ImageRepo, foobar.Tag)

			ORAS("restore", "--input", backupDir, Flags.ExcludeReferrers, "--dry-run", dstRef).
				MatchKeyWords("would push tag", srcRef, restoredRef, "0 referrer(s)").
				MatchKeyWords("Dry run complete", "1 tag(s) would be restored").
				Exec()

			// Verify nothing was actually pushed
			ORAS("manifest", "fetch", restoredRef).
				ExpectFailure().
				MatchErrKeyWords("Error response from registry:", "not found").
				Exec()
		})

		It("should simulate restore with referrers", func() {
			// Create a backup with referrers
			tmpDir := GinkgoT().TempDir()
			backupDir := filepath.Join(tmpDir, "backup-dry-run-referrers")
			srcRef := RegistryRef(ZOTHost, ArtifactRepo, foobar.Tag)

			ORAS("backup", "--output", backupDir, Flags.IncludeReferrers, srcRef).Exec()

			// Create target repo for restore
			testRepo := restoreTestRepo("restore-dry-run-referrers")
			dstRef := fmt.Sprintf("%s/%s", ZOTHost, testRepo)
			restoredRef := RegistryRef(ZOTHost, testRepo+"/"+ArtifactRepo, foobar.Tag)

			ORAS("restore", "--input", backupDir, "--dry-run", dstRef).
				MatchKeyWords("would push tag", srcRef, restoredRef, "2 referrer(s)").
				MatchKeyWords("Dry run complete", "1 tag(s) would be restored").
				Exec()

			// Verify nothing was actually pushed
			ORAS("manifest", "fetch", restoredRef).
				ExpectFailure().
				MatchErrKeyWords("Error response from registry:", "not found").
				Exec()
		})
	})

	When("using --concurrency", func() {
		It("should successfully restore with custom concurrency level when --concurrency flag is used", func() {
			// Create a backup
			tmpDir := GinkgoT().TempDir()
			backupDir := filepath.Join(tmpDir, "backup-concurrency")
			srcRef := RegistryRef(ZOTHost, ImageRepo, ma.Tag)

			ORAS("backup", "--output", backupDir, srcRef).Exec()

			// Create target repo for restore
			testRepo := restoreTestRepo("restore-concurrency")
			// the backup records the source repository, which is restored under the target
			restoredRepo := testRepo + "/" + ImageRepo
			dstRef := fmt.Sprintf("%s/%s", ZOTHost, testRepo)

			stateKeys := ma.IndexStateKeys
			ORAS("restore", "--input", backupDir, "--concurrency", "1", dstRef).
				MatchStatus(stateKeys, true, len(stateKeys)).
				Exec()

			// Verify restored content
			CompareRef(srcRef, RegistryRef(ZOTHost, restoredRepo, ma.Tag))
		})
	})

	When("using --distribution-spec options", func() {

		It("should restore using referrers API with --distribution-spec v1.1-referrers-api", func() {
			tmpDir := GinkgoT().TempDir()
			backupDir := filepath.Join(tmpDir, "backup-referrers-api")
			srcRef := RegistryRef(ZOTHost, ArtifactRepo, foobar.Tag)

			ORAS("backup", "--output", backupDir, Flags.IncludeReferrers, srcRef).Exec()

			// Create target repo for restore
			testRepo := restoreTestRepo("restore-referrers-api")
			// the backup records the source repository, which is restored under the target
			restoredRepo := testRepo + "/" + ArtifactRepo
			dstRef := fmt.Sprintf("%s/%s", ZOTHost, testRepo)

			ORAS("restore", "--input", backupDir, Flags.DistributionSpec, "v1.1-referrers-api", dstRef).
				MatchKeyWords("2 referrer(s)").
				Exec()

			// Verify restored content
			CompareRef(srcRef, RegistryRef(ZOTHost, restoredRepo, foobar.Tag))

			// Verify referrers were restored
			referrers := ORAS("discover", RegistryRef(ZOTHost, restoredRepo, foobar.Tag), "--format", "go-template={{range .referrers}}{{println .digest}}{{end}}").Exec().Out.Contents()
			for referrerDgst := range strings.SplitSeq(strings.TrimSpace(string(referrers)), "\n") {
				CompareRef(RegistryRef(ZOTHost, ArtifactRepo, referrerDgst), RegistryRef(ZOTHost, restoredRepo, referrerDgst))
			}
		})

		It("should restore using tag schema with --distribution-spec v1.1-referrers-tag", func() {
			tmpDir := GinkgoT().TempDir()
			backupDir := filepath.Join(tmpDir, "backup-referrers-tag")
			srcRef := RegistryRef(FallbackHost, ArtifactRepo, foobar.Tag)

			ORAS("backup", "--output", backupDir, Flags.IncludeReferrers, srcRef).Exec()

			// Create target repo for restore
			testRepo := restoreTestRepo("restore-referrers-tag")
			// the backup records the source repository, which is restored under the target
			restoredRepo := testRepo + "/" + ArtifactRepo
			dstRef := fmt.Sprintf("%s/%s", FallbackHost, testRepo)

			ORAS("restore", "--input", backupDir, Flags.DistributionSpec, "v1.1-referrers-tag", dstRef).
				MatchKeyWords("2 referrer(s)").
				Exec()

			// Verify restored content
			CompareRef(srcRef, RegistryRef(FallbackHost, restoredRepo, foobar.Tag))

			// Verify referrers were restored
			referrers := ORAS("discover", RegistryRef(FallbackHost, restoredRepo, foobar.Tag), "--format", "go-template={{range .referrers}}{{println .digest}}{{end}}").Exec().Out.Contents()
			for referrerDgst := range strings.SplitSeq(strings.TrimSpace(string(referrers)), "\n") {
				CompareRef(RegistryRef(FallbackHost, ArtifactRepo, referrerDgst), RegistryRef(FallbackHost, restoredRepo, referrerDgst))
			}
		})
	})

	When("restoring a backup of several repositories", func() {
		// newMultiRepoBackup creates a backup holding the foobar image under each of the given references.
		newMultiRepoBackup := func(name string, refs ...string) string {
			backupDir := filepath.Join(GinkgoT().TempDir(), name)
			srcRef := RegistryRef(ZOTHost, ImageRepo, foobar.Tag)
			for _, ref := range refs {
				ORAS("cp", srcRef, ref, Flags.ToLayoutPath, backupDir).Exec()
			}
			return backupDir
		}

		It("should restore every repository to a registry", func() {
			ns := restoreTestRepo("multi-repo-to-registry")
			srcRepos := []string{ns + "/a", ns + "/b"}
			backupDir := newMultiRepoBackup("backup-multi-repo",
				"src.example.com/"+srcRepos[0]+":"+foobar.Tag,
				"src.example.com/"+srcRepos[1]+":"+foobar.Tag)

			ORAS("restore", "--input", backupDir, Flags.ExcludeReferrers, ZOTHost).
				MatchKeyWords("Successfully restored 2 tag(s)").
				Exec()

			// each source repository is restored under the registry, with the same tag
			srcRef := RegistryRef(ZOTHost, ImageRepo, foobar.Tag)
			for _, repo := range srcRepos {
				CompareRef(srcRef, RegistryRef(ZOTHost, repo, foobar.Tag))
			}
		})

		It("should fail before pushing when two source repositories restore to the same repository", func() {
			testRepo := restoreTestRepo("multi-repo-collision")
			backupDir := newMultiRepoBackup("backup-collision",
				"a.example.com/app:"+foobar.Tag,
				"b.example.com/app:"+foobar.Tag)
			dstRef := fmt.Sprintf("%s/%s", ZOTHost, testRepo)

			for _, dryRun := range [][]string{nil, {"--dry-run"}} {
				args := append([]string{"restore", "--input", backupDir, Flags.ExcludeReferrers, dstRef}, dryRun...)
				ORAS(args...).ExpectFailure().
					MatchErrKeyWords("Error:", `"a.example.com/app" and "b.example.com/app" both restore to`, "--preserve-source-registry").
					Exec()
			}

			// Verify nothing was pushed
			ORAS("manifest", "fetch", RegistryRef(ZOTHost, testRepo+"/app", foobar.Tag)).
				ExpectFailure().
				MatchErrKeyWords("Error response from registry:", "not found").
				Exec()
		})

		It("should keep the source registry in the repository path with --preserve-source-registry", func() {
			testRepo := restoreTestRepo("multi-repo-preserve-source-registry")
			backupDir := newMultiRepoBackup("backup-preserve-source-registry",
				"a.example.com/app:"+foobar.Tag,
				"b.example.com/app:"+foobar.Tag)
			dstRef := fmt.Sprintf("%s/%s", ZOTHost, testRepo)

			ORAS("restore", "--input", backupDir, Flags.ExcludeReferrers, "--preserve-source-registry", dstRef).
				MatchKeyWords("Successfully restored 2 tag(s)").
				Exec()

			srcRef := RegistryRef(ZOTHost, ImageRepo, foobar.Tag)
			for _, srcRegistry := range []string{"a.example.com", "b.example.com"} {
				CompareRef(srcRef, RegistryRef(ZOTHost, testRepo+"/"+srcRegistry+"/app", foobar.Tag))
			}
		})

		It("should encode a source registry with a port with --preserve-source-registry", func() {
			srcRef := RegistryRef(ZOTHost, ImageRepo, foobar.Tag)
			backupDir := filepath.Join(GinkgoT().TempDir(), "backup-preserve-source-registry-port")
			ORAS("backup", "--output", backupDir, srcRef).Exec()

			testRepo := restoreTestRepo("preserve-source-registry-port")
			dstRef := fmt.Sprintf("%s/%s", ZOTHost, testRepo)
			// the source registry host is lowercased and its port ':' is replaced by '_'
			encodedHost := strings.ReplaceAll(strings.ToLower(ZOTHost), ":", "_")
			restoredRef := RegistryRef(ZOTHost, testRepo+"/"+encodedHost+"/"+ImageRepo, foobar.Tag)

			ORAS("restore", "--input", backupDir, Flags.ExcludeReferrers, "--preserve-source-registry", "--dry-run", dstRef).
				MatchKeyWords("would push tag", srcRef, restoredRef).
				Exec()

			ORAS("restore", "--input", backupDir, Flags.ExcludeReferrers, "--preserve-source-registry", dstRef).
				MatchKeyWords("Successfully restored 1 tag(s)").
				Exec()

			CompareRef(srcRef, restoredRef)
		})
	})

	When("handling error cases", func() {
		It("should fail when restoring from a non-existent backup directory", func() {
			// Create a non-existent backup directory
			nonExistentBackupDir := filepath.Join(GinkgoT().TempDir(), "non-existent-backup")

			dstRef := fmt.Sprintf("%s/%s", ZOTHost, restoreTestRepo("restore-non-existent-backup"))
			// Attempt to restore from non-existent backup directory
			ORAS("restore", "--input", nonExistentBackupDir, dstRef).ExpectFailure().
				MatchErrKeyWords("Error:", "no such file or directory").
				Exec()
		})

		It("should fail when restoring from an empty backup directory", func() {
			// Create an invalid backup directory (empty)
			tmpDir := GinkgoT().TempDir()
			backupDir := filepath.Join(tmpDir, "empty-backup")
			err := os.MkdirAll(backupDir, 0755)
			Expect(err).ToNot(HaveOccurred())

			dstRef := fmt.Sprintf("%s/%s", ZOTHost, restoreTestRepo("restore-empty-backup"))
			// Attempt to restore from empty backup directory
			ORAS("restore", "--input", backupDir, dstRef).ExpectFailure().
				MatchErrKeyWords("Error:", "no tags found").
				MatchErrKeyWords("oras repo tags --oci-layout"). // test recommendation
				Exec()
		})

		It("should fail when restoring from an invalid backup directory", func() {
			// Create an invalid backup directory (not a valid OCI layout)
			tmpDir := GinkgoT().TempDir()
			backupDir := filepath.Join(tmpDir, "invalid-backup")
			err := os.MkdirAll(backupDir, 0755)
			Expect(err).ToNot(HaveOccurred())
			err = os.WriteFile(filepath.Join(backupDir, "invalid-file.txt"), []byte("not a valid layout"), 0644)
			Expect(err).ToNot(HaveOccurred())

			dstRef := fmt.Sprintf("%s/%s", ZOTHost, restoreTestRepo("restore-invalid-backup"))
			// Attempt to restore from invalid backup directory
			ORAS("restore", "--input", backupDir, dstRef).ExpectFailure().
				MatchErrKeyWords("Error:", "no tags found").
				Exec()
		})

		It("should fail when restoring from a non-existent backup tar file", func() {
			// Create a non-existent backup tar file
			nonExistentBackupTar := filepath.Join(GinkgoT().TempDir(), "non-existent-backup.tar")

			dstRef := fmt.Sprintf("%s/%s", ZOTHost, restoreTestRepo("restore-non-existent-backup-tar"))
			// Attempt to restore from non-existent backup tar file
			ORAS("restore", "--input", nonExistentBackupTar, dstRef).ExpectFailure().
				MatchErrKeyWords("Error:", "no such file or directory").
				Exec()
		})

		It("should fail when restoring from an invalid backup tar file", func() {
			// Create an invalid backup tar file (empty)
			tmpDir := GinkgoT().TempDir()
			backupTar := filepath.Join(tmpDir, "invalid-backup.tar")

			// Create an empty file to simulate an invalid tar
			err := os.WriteFile(backupTar, []byte{}, 0644)
			Expect(err).ToNot(HaveOccurred())

			dstRef := fmt.Sprintf("%s/%s", ZOTHost, restoreTestRepo("restore-invalid-backup-tar"))
			// Attempt to restore from invalid backup tar file
			ORAS("restore", "--input", backupTar, dstRef).ExpectFailure().
				MatchErrKeyWords("Error:", "invalid OCI Image Layout").
				Exec()
		})

		It("should fail when restoring from a non-tar file", func() {
			// Create a non-tar backup file
			tmpDir := GinkgoT().TempDir()
			backupFile := filepath.Join(tmpDir, "not-tar.txt")

			// Create an empty non-tar file
			err := os.WriteFile(backupFile, []byte{}, 0644)
			Expect(err).ToNot(HaveOccurred())

			dstRef := fmt.Sprintf("%s/%s", ZOTHost, restoreTestRepo("restore-non-tar-file"))
			// Attempt to restore from invalid backup tar file
			ORAS("restore", "--input", backupFile, dstRef).ExpectFailure().
				MatchErrKeyWords("Error:").
				Exec()
		})

		It("should fail when no tags are found in the backup", func() {
			// Create an OCI layout with no tags
			tmpDir := GinkgoT().TempDir()
			backupDir := filepath.Join(tmpDir, "backup-no-tags")
			ORAS("push", Flags.Layout, backupDir).Exec()

			dstRef := fmt.Sprintf("%s/%s", ZOTHost, restoreTestRepo("restore-no-tags"))
			// Attempt to restore from empty backup
			ORAS("restore", "--input", backupDir, dstRef).ExpectFailure().
				MatchErrKeyWords("Error:", "no tags found").
				MatchErrKeyWords("oras repo tags --oci-layout"). // test recommendation
				Exec()
		})

		It("should fail when a tag is specified in the target", func() {
			// Create a backup
			tmpDir := GinkgoT().TempDir()
			backupDir := filepath.Join(tmpDir, "backup-tag-in-target")
			srcRef := RegistryRef(ZOTHost, ImageRepo, foobar.Tag)

			ORAS("backup", "--output", backupDir, srcRef).Exec()

			// The target only names where artifacts go, so a tag in it is rejected
			testRepo := restoreTestRepo("restore-tag-in-target")
			dstRef := RegistryRef(ZOTHost, testRepo, foobar.Tag)
			ORAS("restore", "--input", backupDir, dstRef).ExpectFailure().
				MatchErrKeyWords("Error:", "a tag or digest cannot be specified in the restore target").Exec()

			// Verify nothing was pushed
			ORAS("manifest", "fetch", RegistryRef(ZOTHost, testRepo+"/"+ImageRepo, foobar.Tag)).
				ExpectFailure().
				MatchErrKeyWords("Error response from registry:", "not found").
				Exec()
		})

		It("should fail when trying to restore to a non-existent registry", func() {
			// Create a backup
			tmpDir := GinkgoT().TempDir()
			backupDir := filepath.Join(tmpDir, "backup-nonexistent-registry")
			srcRef := RegistryRef(ZOTHost, ImageRepo, foobar.Tag)

			ORAS("backup", "--output", backupDir, srcRef).Exec()

			// Try to restore to a non-existent registry
			nonExistentRegistry := fmt.Sprintf("non-existent-repo-%d-%d.example.com", GinkgoRandomSeed(), time.Now().UnixNano())
			dstRef := fmt.Sprintf("%s/%s", nonExistentRegistry, restoreTestRepo("restore-test"))

			// Attempt to restore to a non-existent registry
			ORAS("restore", "--input", backupDir, dstRef).ExpectFailure().
				MatchErrKeyWords("Error:").
				Exec()
		})
	})
})
