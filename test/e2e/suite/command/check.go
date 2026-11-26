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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	ma "oras.land/oras/test/e2e/internal/testdata/multi_arch"
	. "oras.land/oras/test/e2e/internal/utils"
)

// blobFile returns the path of a blob in the OCI layout at root.
func blobFile(root, dgst string) string {
	algorithm, encoded, _ := strings.Cut(dgst, ":")
	return filepath.Join(root, "blobs", algorithm, encoded)
}

// replaceBlob replaces the content of a blob in the OCI layout at root. The
// blobs of a prepared layout are hard links to the test data, so the file is
// removed first; writing through the link would change the test data.
func replaceBlob(root, dgst string, content []byte) {
	path := blobFile(root, dgst)
	Expect(os.Remove(path)).To(Succeed())
	Expect(os.WriteFile(path, content, 0o644)).To(Succeed())
}

func checkTestRepo(text string) string {
	return fmt.Sprintf("command/check/%d/%s", GinkgoRandomSeed(), text)
}

var _ = Describe("ORAS beginners:", func() {
	When("running check command", func() {
		It("should show help description with the experimental notice", func() {
			ORAS("check", "--help").
				MatchKeyWords("[Experimental]", ExampleDesc, "--concurrency", "--platform", "--oci-layout").
				Exec()
		})

		It("should fail when no artifact reference is given", func() {
			ORAS("check").ExpectFailure().MatchErrKeyWords("Error:", "the artifact reference to check").Exec()
		})

		It("should fail when the artifact reference has no tag or digest", func() {
			ORAS("check", Flags.Layout, PrepareTempOCI(ArtifactRepo)).ExpectFailure().
				MatchErrKeyWords("Error:", "no tag or digest specified").Exec()
		})

		It("should fail when the layout path does not exist", func() {
			ORAS("check", Flags.Layout, LayoutRef(filepath.Join(GinkgoT().TempDir(), "missing"), foobar.Tag)).ExpectFailure().
				MatchErrKeyWords("Error:", "invalid argument").Exec()
		})

		It("should fail when the tag is not in the layout", func() {
			ORAS("check", Flags.Layout, LayoutRef(PrepareTempOCI(ArtifactRepo), InvalidTag)).ExpectFailure().
				MatchErrKeyWords("Error:", "failed to resolve").Exec()
		})
	})
})

var _ = Describe("OCI image layout users:", func() {
	When("running check command", func() {
		It("should verify an intact artifact and print a line for each node", func() {
			root := PrepareTempOCI(ArtifactRepo)
			ORAS("check", Flags.Layout, LayoutRef(root, foobar.Tag)).
				MatchKeyWords(
					"Verified", foobar.ManifestStateKey.Digest, foobar.FileBarStateKey.Digest,
					"Checked 4 nodes: 4 verified, 0 failed",
				).Exec()
		})

		It("should verify an artifact by digest", func() {
			root := PrepareTempOCI(ArtifactRepo)
			ORAS("check", Flags.Layout, LayoutRef(root, foobar.Digest)).
				MatchKeyWords("Checked 4 nodes: 4 verified, 0 failed").Exec()
		})

		It("should verify a whole multi-platform artifact", func() {
			root := PrepareTempOCI(ImageRepo)
			ORAS("check", Flags.Layout, LayoutRef(root, ma.Tag)).
				MatchKeyWords("Checked 8 nodes: 8 verified, 0 failed").Exec()
		})

		It("should verify one platform of a multi-platform artifact", func() {
			root := PrepareTempOCI(ImageRepo)
			ORAS("check", Flags.Layout, LayoutRef(root, ma.Tag), "--platform", "linux/amd64").
				MatchKeyWords("Checked 3 nodes: 3 verified, 0 failed").Exec()
		})

		It("should verify a manifest that is only nested in an index", func() {
			root := PrepareTempOCI(ImageRepo)
			// The linux/amd64 manifest is a member of the multi-platform index, so
			// it has to be found in the blob storage of the layout.
			ORAS("check", Flags.Layout, LayoutRef(root, string(ma.LinuxAMD64.Digest))).
				MatchKeyWords("Checked 3 nodes: 3 verified, 0 failed").Exec()
		})

		It("should fail and name the corrupt blob when a layer was changed", func() {
			root := PrepareTempOCI(ArtifactRepo)
			replaceBlob(root, foobar.BarBlobDigest, []byte("baz")) // same size as "bar"
			ORAS("check", Flags.Layout, LayoutRef(root, foobar.Tag)).ExpectFailure().
				MatchKeyWords(
					"Failed", "Checked 4 nodes: 3 verified, 1 failed",
					"Corrupt:", foobar.BarBlobDigest,
				).
				MatchErrKeyWords("Error:", "check failed: 1 of 4 nodes did not pass").Exec()
		})

		It("should fail and name the missing blob when a layer was removed", func() {
			root := PrepareTempOCI(ArtifactRepo)
			Expect(os.Remove(blobFile(root, foobar.BarBlobDigest))).To(Succeed())
			ORAS("check", Flags.Layout, LayoutRef(root, foobar.Tag)).ExpectFailure().
				MatchKeyWords(
					"Checked 4 nodes: 3 verified, 1 failed",
					"Missing:", foobar.BarBlobDigest,
				).
				MatchErrKeyWords("Error:", "check failed: 1 of 4 nodes did not pass").Exec()
		})

		It("should report every problem, and not stop at the first one", func() {
			root := PrepareTempOCI(ArtifactRepo)
			Expect(os.Remove(blobFile(root, foobar.BarBlobDigest))).To(Succeed())
			replaceBlob(root, foobar.FooBlobDigest, []byte("fox"))
			ORAS("check", Flags.Layout, LayoutRef(root, foobar.Tag)).ExpectFailure().
				MatchKeyWords(
					"Checked 4 nodes: 2 verified, 2 failed",
					"Missing:", foobar.BarBlobDigest,
					"Corrupt:", foobar.FooBlobDigest,
				).
				MatchErrKeyWords("check failed: 2 of 4 nodes did not pass").Exec()
		})
	})
})

var _ = Describe("Common registry users:", func() {
	When("running check command", func() {
		It("should verify an artifact in a registry by tag and by digest", func() {
			repo := checkTestRepo("intact")
			root := PrepareTempOCI(ArtifactRepo)
			ORAS("cp", Flags.FromLayout, LayoutRef(root, foobar.Tag), RegistryRef(ZOTHost, repo, foobar.Tag)).Exec()

			ORAS("check", RegistryRef(ZOTHost, repo, foobar.Tag)).
				MatchKeyWords("Checked 4 nodes: 4 verified, 0 failed").Exec()
			ORAS("check", RegistryRef(ZOTHost, repo, foobar.Digest)).
				MatchKeyWords("Checked 4 nodes: 4 verified, 0 failed").Exec()
		})

		It("should verify one platform of a multi-platform artifact in a registry", func() {
			repo := checkTestRepo("platform")
			root := PrepareTempOCI(ImageRepo)
			ORAS("cp", "-r", Flags.FromLayout, LayoutRef(root, ma.Tag), RegistryRef(ZOTHost, repo, ma.Tag)).Exec()

			ORAS("check", RegistryRef(ZOTHost, repo, ma.Tag)).
				MatchKeyWords("Checked 8 nodes: 8 verified, 0 failed").Exec()
			ORAS("check", RegistryRef(ZOTHost, repo, ma.Tag), "--platform", "linux/arm64").
				MatchKeyWords("Checked 3 nodes: 3 verified, 0 failed").Exec()
		})

		It("should fail when the tag is not in the registry", func() {
			ORAS("check", RegistryRef(ZOTHost, ArtifactRepo, InvalidTag)).ExpectFailure().
				MatchErrKeyWords(RegistryErrorPrefix, "failed to resolve").Exec()
		})

		It("should fail when the repository name is invalid", func() {
			ORAS("check", ZOTHost+"/"+InvalidRepo+":"+foobar.Tag).ExpectFailure().
				MatchErrKeyWords("Error:", "invalid reference").Exec()
		})
	})
})
