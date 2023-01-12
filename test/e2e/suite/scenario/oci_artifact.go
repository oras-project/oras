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

package scenario

import (
	"fmt"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	. "oras.land/oras/test/e2e/internal/utils"
	"oras.land/oras/test/e2e/internal/utils/match"
)

var _ = Describe("Common OCI artifact users:", Ordered, func() {
	repo := "scenario/oci-artifact"
	When("pushing images and attaching", func() {
		tag := "artifact"
		var tempDir string
		BeforeAll(func() {
			tempDir = GinkgoT().TempDir()
			if err := CopyTestData(tempDir); err != nil {
				panic(err)
			}
		})

		pulledManifest := "packed.json"
		pullRoot := "pulled"
		It("should push and pull an artifact", func() {
			ORAS("push", Reference(Host, repo, tag), "--artifact-type", "test-artifact", blobFileNames[0], blobFileNames[1], blobFileNames[2], "-v", "--export-manifest", pulledManifest).
				MatchStatus(pushFileStateKeys, true, 3).
				WithWorkDir(tempDir).
				WithDescription("push with manifest exported").Exec()

			fetched := ORAS("manifest", "fetch", Reference(Host, repo, tag)).Exec()
			MatchFile(filepath.Join(tempDir, pulledManifest), string(fetched.Out.Contents()), DefaultTimeout)

			ORAS("pull", Reference(Host, repo, tag), "-v", "-o", pullRoot).
				MatchStatus(pushFileStateKeys, true, 3).
				WithWorkDir(tempDir).
				WithDescription("pull artFiles with config").Exec()

			for _, f := range blobFileNames {
				Binary("diff", filepath.Join(f), filepath.Join(pullRoot, f)).
					WithWorkDir(tempDir).
					WithDescription("download identical file " + f).Exec()
			}
		})

		It("should attach and pull an artifact", func() {
			subject := Reference(Host, repo, tag)
			ORAS("attach", subject, "--artifact-type", "test-artifact", fmt.Sprint(attachFileName, ":", attachFileMedia), "-v", "--export-manifest", pulledManifest).
				MatchStatus([]match.StateKey{attachFileStateKey}, true, 1).
				WithWorkDir(tempDir).
				WithDescription("attach with manifest exported").Exec()

			session := ORAS("discover", subject, "-o", "json").Exec()
			digest := string(Binary("jq", "-r", ".manifests[].digest").WithInput(session.Out).Exec().Out.Contents())
			fetched := ORAS("manifest", "fetch", Reference(Host, repo, digest)).MatchKeyWords(attachFileMedia).Exec()
			MatchFile(filepath.Join(tempDir, pulledManifest), string(fetched.Out.Contents()), DefaultTimeout)

			ORAS("pull", Reference(Host, repo, digest), "-v", "-o", pullRoot).
				MatchStatus([]match.StateKey{attachFileStateKey}, true, 1).
				WithWorkDir(tempDir).
				WithDescription("pull attached artifact").Exec()
			Binary("diff", filepath.Join(attachFileName), filepath.Join(pullRoot, attachFileName)).
				WithWorkDir(tempDir).
				WithDescription("download identical file " + attachFileName).Exec()

			ORAS("attach", subject, "--artifact-type", "test-artifact", fmt.Sprint(attachFileName, ":", attachFileMedia), "-v", "--export-manifest", pulledManifest).
				MatchStatus([]match.StateKey{attachFileStateKey}, true, 1).
				WithWorkDir(tempDir).
				WithDescription("attach again with manifest exported").Exec()

			session = ORAS("discover", subject, "-o", "json").Exec()
			raw := Binary("jq", "-r", ".manifests[].digest").WithInput(session.Out).Exec().Out.Contents()
			digests := strings.Split(strings.TrimSpace(string(raw)), "\n")
			gomega.Expect(len(digests)).To(gomega.Equal(2))
			digest = strings.TrimSpace(digest)
			if digests[0] != digest {
				digest = digests[0]
			}
			fetched = ORAS("manifest", "fetch", Reference(Host, repo, digest)).MatchKeyWords(attachFileMedia).Exec()
			MatchFile(filepath.Join(tempDir, pulledManifest), string(fetched.Out.Contents()), DefaultTimeout)

			ORAS("pull", Reference(Host, repo, string(digest)), "-v", "-o", pullRoot, "--include-subject").
				MatchStatus(append(pushFileStateKeys, attachFileStateKey), true, 4).
				WithWorkDir(tempDir).
				WithDescription("pull attached artifact and subject").Exec()

			for _, f := range append(blobFileNames, attachFileName) {
				Binary("diff", filepath.Join(f), filepath.Join(pullRoot, f)).
					WithWorkDir(tempDir).
					WithDescription("download identical file " + f).Exec()
			}
		})
	})
})
