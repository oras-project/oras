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
	"os"
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"oras.land/oras-go/v2"
	"oras.land/oras/test/e2e/internal/testdata/feature"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	ma "oras.land/oras/test/e2e/internal/testdata/multi_arch"
	. "oras.land/oras/test/e2e/internal/utils"
)

func verifyBackupDirectoryStructure(backupPath string) {
	Expect(backupPath).To(BeADirectory())
	Expect(filepath.Join(backupPath, "ingest")).ToNot(BeAnExistingFile())
}

func verifyBackupTarStructure(tarPath string) {
	Expect(tarPath).To(BeAnExistingFile())
	Expect(tarPath).NotTo(BeADirectory())
}

func compareBackupRef(srcRef, dstRef string) {
	srcManifest := ORAS("manifest", "fetch", srcRef).WithDescription("fetch from source to validate").Exec().Out.Contents()
	dstManifest := ORAS("manifest", "fetch", Flags.Layout, dstRef).WithDescription("fetch from destination OCI layout to validate").Exec().Out.Contents()
	Expect(srcManifest).To(Equal(dstManifest))
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

		It("should fail when source doesn't exist", func() {
			tmpDir := GinkgoT().TempDir()
			defer os.RemoveAll(tmpDir)

			ORAS("backup", "--output", tmpDir, RegistryRef(ZOTHost, ImageRepo, InvalidTag)).ExpectFailure().
				MatchErrKeyWords("Error:", InvalidTag).Exec()
		})
	})
})

var _ = Describe("ORAS users:", func() {
	When("backing up a single tag to directory", func() {
		It("should successfully backup an image to a directory", func() {
			tmpDir := GinkgoT().TempDir()
			outDir := filepath.Join(tmpDir, "backup-single-tag")
			srcRef := RegistryRef(ZOTHost, ImageRepo, foobar.Tag)
			foobarStates := append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageConfigStateKey(oras.MediaTypeUnknownConfig))

			ORAS("backup", "--output", outDir, srcRef).
				MatchStatus(foobarStates, true, len(foobarStates)).
				Exec()

			// Verify backup output structure
			verifyBackupDirectoryStructure(outDir)

			// Verify backed up content
			dstRef := LayoutRef(outDir, foobar.Tag)
			compareBackupRef(srcRef, dstRef)
		})

		It("should backup an artifact with its referrers to a directory", func() {
			tmpDir := GinkgoT().TempDir()
			outDir := filepath.Join(tmpDir, "backup-single-tag-referrers")
			// Using artifact from ZOT registry that has referrers
			srcRef := RegistryRef(ZOTHost, ArtifactRepo, foobar.Tag)
			foobarStates := append(append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageReferrerConfigStateKeys[0]), foobar.ImageReferrersStateKeys...)

			ORAS("backup", "--output", outDir, Flags.IncludeReferrers, srcRef).
				MatchStatus(foobarStates, true, len(foobarStates)).
				Exec()

			// Verify backup output structure
			verifyBackupDirectoryStructure(outDir)

			// Verify backed up content
			dstRef := LayoutRef(outDir, foobar.Tag)
			compareBackupRef(srcRef, dstRef)
			referrers := ORAS("discover", Flags.Layout, dstRef, "--format", "go-template={{range .referrers}}{{println .digest}}{{end}}").Exec().Out.Contents()
			for referrerDgst := range strings.SplitSeq(strings.TrimSpace(string(referrers)), "\n") {
				compareBackupRef(RegistryRef(ZOTHost, ArtifactRepo, referrerDgst), LayoutRef(outDir, referrerDgst))
			}
		})

		It("should successfully backup a multi-arch artifact to a directory", func() {
			tmpDir := GinkgoT().TempDir()
			outDir := filepath.Join(tmpDir, "backup-multi-arch")
			srcRef := RegistryRef(ZOTHost, ArtifactRepo, ma.Tag)
			stateKeys := ma.IndexStateKeys

			ORAS("backup", "--output", outDir, srcRef).
				MatchStatus(stateKeys, true, len(stateKeys)).
				Exec()

			// Verify backup output structure
			verifyBackupDirectoryStructure(outDir)

			// Verify backed up content
			dstRef := LayoutRef(outDir, ma.Tag)
			compareBackupRef(srcRef, dstRef)
		})

		It("should successfully backup a multi-arch artifact with referrers to a directory", func() {
			tmpDir := GinkgoT().TempDir()
			outDir := filepath.Join(tmpDir, "backup-multi-arch-referrers")
			srcRef := RegistryRef(ZOTHost, ArtifactRepo, ma.Tag)
			stateKeys := append(ma.IndexStateKeys, ma.IndexZOTReferrerStateKey, ma.LinuxAMD64ReferrerConfigStateKey)

			ORAS("backup", "--output", outDir, Flags.IncludeReferrers, srcRef).
				MatchStatus(stateKeys, true, len(stateKeys)).
				Exec()

			// Verify backup output structure
			verifyBackupDirectoryStructure(outDir)

			// Verify backed up content
			dstRef := LayoutRef(outDir, ma.Tag)
			compareBackupRef(srcRef, dstRef)
			referrers := ORAS("discover", Flags.Layout, dstRef, "--format", "go-template={{range .referrers}}{{println .digest}}{{end}}").Exec().Out.Contents()
			for referrerDgst := range strings.SplitSeq(strings.TrimSpace(string(referrers)), "\n") {
				compareBackupRef(RegistryRef(ZOTHost, ArtifactRepo, referrerDgst), LayoutRef(outDir, referrerDgst))
			}
		})
	})

	When("backing up a single tag to tar archive", func() {
		It("should successfully backup an image to tar file", func() {
			tmpDir := GinkgoT().TempDir()
			outTar := filepath.Join(tmpDir, "backup.tar")
			src := RegistryRef(ZOTHost, ImageRepo, foobar.Tag)
			foobarStates := append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageConfigStateKey(oras.MediaTypeUnknownConfig))

			ORAS("backup", "--output", outTar, src).
				MatchStatus(foobarStates, true, len(foobarStates)).
				Exec()

			// Verify backup structure
			verifyBackupTarStructure(outTar)

			// Verify backed up content
			dstRef := LayoutRef(outTar, foobar.Tag)
			compareBackupRef(src, dstRef)
		})

		It("should successfully backup a multi-arch artifact to tar file", func() {
			tmpDir := GinkgoT().TempDir()
			outTar := filepath.Join(tmpDir, "multi-arch.tar")
			src := RegistryRef(ZOTHost, ImageRepo, ma.Tag)
			stateKeys := ma.IndexStateKeys

			ORAS("backup", "--output", outTar, src).
				MatchStatus(stateKeys, true, len(stateKeys)).
				Exec()

			// Verify backup structure
			verifyBackupTarStructure(outTar)

			// Verify backed up content
			dstRef := LayoutRef(outTar, ma.Tag)
			compareBackupRef(src, dstRef)
		})
	})

	// 	When("using --concurrency flag", func() {
	// 		It("should successfully backup with custom concurrency level", func() {
	// 			outDir := filepath.Join(tmpDir, "concurrency")
	// 			src := RegistryRef(ZOTHost, ImageRepo, ma.Tag)

	// 			ORAS("backup", "--output", outDir, "--concurrency", "5", src).
	// 				MatchStatus(ma.StateKeys, true, len(ma.StateKeys)).
	// 				Exec()

	// 			// Verify directory structure was created correctly
	// 			Expect(outDir).To(BeADirectory())
	// 			Expect(filepath.Join(outDir, "index.json")).To(BeAnExistingFile())
	// 			Expect(filepath.Join(outDir, "blobs")).To(BeADirectory())
	// 		})
	// 	})

	// 	When("backing up multiple tags", func() {
	// 		It("should backup all specified tags", func() {
	// 			outDir := filepath.Join(tmpDir, "multiple-tags")
	// 			repo := ZOTHost + "/" + ArtifactRepo

	// 			// Specify multiple tags in format: repo:tag1,tag2
	// 			ORAS("backup", "--output", outDir, fmt.Sprintf("%s:%s,%s", repo, foobar.Tag, "multi")).
	// 				Exec()

	// 			// Verify directory structure was created correctly
	// 			Expect(outDir).To(BeADirectory())
	// 			Expect(filepath.Join(outDir, "index.json")).To(BeAnExistingFile())
	// 			Expect(filepath.Join(outDir, "blobs")).To(BeADirectory())
	// 		})

	// 		It("should auto-discover all tags when no tag is specified", func() {
	// 			outDir := filepath.Join(tmpDir, "discover-tags")
	// 			repo := ZOTHost + "/" + ArtifactRepo

	// 			// Don't specify any tag - should discover all tags
	// 			ORAS("backup", "--output", outDir, repo).Exec()

	// 			// Verify directory structure was created correctly
	// 			Expect(outDir).To(BeADirectory())
	// 			Expect(filepath.Join(outDir, "index.json")).To(BeAnExistingFile())
	// 			Expect(filepath.Join(outDir, "blobs")).To(BeADirectory())
	// 		})
	// 	})

	// 	When("using insecure registries", func() {
	// 		It("should successfully backup from insecure registry using --insecure flag", func() {
	// 			outDir := filepath.Join(tmpDir, "insecure")
	// 			src := RegistryRef(RegistryHost, ArtifactRepo, foobar.Tag)

	// 			ORAS("backup", "--output", outDir, "--insecure", src).Exec()

	// 			// Verify directory structure was created correctly
	// 			Expect(outDir).To(BeADirectory())
	// 			Expect(filepath.Join(outDir, "index.json")).To(BeAnExistingFile())
	// 			Expect(filepath.Join(outDir, "blobs")).To(BeADirectory())
	// 		})

	// 		It("should successfully backup from HTTP registry using --plain-http flag", func() {
	// 			outDir := filepath.Join(tmpDir, "plain-http")
	// 			src := RegistryRef(RegistryFallbackHost, ArtifactRepo, foobar.Tag)

	// 			ORAS("backup", "--output", outDir, "--plain-http", src).Exec()

	// 			// Verify directory structure was created correctly
	// 			Expect(outDir).To(BeADirectory())
	// 			Expect(filepath.Join(outDir, "index.json")).To(BeAnExistingFile())
	// 			Expect(filepath.Join(outDir, "blobs")).To(BeADirectory())
	// 		})
	// 	})
	// })

	// var _ = Describe("ORAS administrators:", func() {
	// 	When("handling error cases", func() {
	// 		It("should fail with appropriate error when invalid reference format provided", func() {
	// 			tmpDir := filepath.Join(os.TempDir(), backupTestDir("invalid-reference"))
	// 			defer os.RemoveAll(tmpDir)

	// 			// Test with a malformed reference
	// 			ORAS("backup", "--output", tmpDir, "invalid/format@sha256:digest").ExpectFailure().
	// 				MatchErrKeyWords("Error:", "digest references are not supported").Exec()
	// 		})

	// 		It("should fail with appropriate error when invalid tag format provided", func() {
	// 			tmpDir := filepath.Join(os.TempDir(), backupTestDir("invalid-tag"))
	// 			defer os.RemoveAll(tmpDir)

	// 			// Test with invalid tag format
	// 			ORAS("backup", "--output", tmpDir, "localhost:5000/repo:invalid@tag").ExpectFailure().
	// 				MatchErrKeyWords("Error:").Exec()
	// 		})

	// 		It("should fail when output directory cannot be created", func() {
	// 			// Create a file that will conflict with our output path
	// 			conflictFile := filepath.Join(os.TempDir(), backupTestDir("conflict-file"))
	// 			defer os.RemoveAll(conflictFile)

	// 			// Create a file (not a directory)
	// 			file, err := os.Create(conflictFile)
	// 			Expect(err).ToNot(HaveOccurred())
	// 			file.Close()

	// 			// Try to use the file as an output directory
	// 			ORAS("backup", "--output", conflictFile, RegistryRef(ZOTHost, ArtifactRepo, foobar.Tag)).
	// 				ExpectFailure().MatchErrKeyWords("Error:").Exec()
	// 		})
	// 	})
})
