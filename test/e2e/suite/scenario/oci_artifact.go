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

	. "github.com/onsi/ginkgo/v2"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	. "oras.land/oras/test/e2e/internal/utils"
	"oras.land/oras/test/e2e/internal/utils/match"
)

var _ = Describe("OCI artifact users:", Ordered, func() {
	repo := "scenario/oci-artifact"
	When("pushing images and attaching", func() {
		tag := "artifact"
		var tempDir string
		BeforeAll(func() {
			tempDir = PrepareTempFiles()
		})

		pulledManifest := "packed.json"
		pullRoot := "pulled"
		It("should push and pull an artifact", func() {
			ORAS("push", RegistryRef(ZOTHost, repo, tag), "--artifact-type", "test/artifact", foobar.FileLayerNames[0], foobar.FileLayerNames[1], foobar.FileLayerNames[2], "-v", "--export-manifest", pulledManifest).
				MatchStatus(foobar.FileStateKeys, true, 3).
				WithWorkDir(tempDir).
				WithDescription("push with manifest exported").Exec()

			fetched := ORAS("manifest", "fetch", RegistryRef(ZOTHost, repo, tag)).Exec()
			MatchFile(filepath.Join(tempDir, pulledManifest), string(fetched.Out.Contents()), DefaultTimeout)

			ORAS("pull", RegistryRef(ZOTHost, repo, tag), "-v", "-o", pullRoot).
				MatchStatus(foobar.FileStateKeys, true, 3).
				WithWorkDir(tempDir).
				WithDescription("pull artFiles with config").Exec()

			for _, f := range foobar.FileLayerNames {
				Binary("diff", filepath.Join(f), filepath.Join(pullRoot, f)).
					WithWorkDir(tempDir).
					WithDescription("download identical file " + f).Exec()
			}
		})

		It("should attach and pull an artifact", func() {
			subject := RegistryRef(ZOTHost, repo, tag)
			ORAS("attach", subject, "--artifact-type", "test/artifact1", fmt.Sprint(foobar.AttachFileName, ":", foobar.AttachFileMedia), "-v", "--export-manifest", pulledManifest).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, true, 1).
				WithWorkDir(tempDir).
				WithDescription("attach with manifest exported").Exec()

			ref := string(ORAS("discover", subject, "--format", "{{(first .manifests).reference}}").Exec().Out.Contents())
			fetched := ORAS("manifest", "fetch", ref).MatchKeyWords(foobar.AttachFileMedia).Exec()
			MatchFile(filepath.Join(tempDir, pulledManifest), string(fetched.Out.Contents()), DefaultTimeout)

			ORAS("pull", ref, "-v", "-o", pullRoot).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, true, 1).
				WithWorkDir(tempDir).
				WithDescription("pull attached artifact").Exec()
			Binary("diff", filepath.Join(foobar.AttachFileName), filepath.Join(pullRoot, foobar.AttachFileName)).
				WithWorkDir(tempDir).
				WithDescription("download identical file " + foobar.AttachFileName).Exec()

			ORAS("attach", subject, "--artifact-type", "test/artifact2", fmt.Sprint(foobar.AttachFileName, ":", foobar.AttachFileMedia), "-v", "--export-manifest", pulledManifest).
				MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, true, 1).
				WithWorkDir(tempDir).
				WithDescription("attach again with manifest exported").Exec()

			ref = string(ORAS("discover", subject, "--format", "{{(first .manifests).reference}}", "--artifact-type", "test/artifact2").Exec().Out.Contents())
			fetched = ORAS("manifest", "fetch", ref).MatchKeyWords(foobar.AttachFileMedia).Exec()
			MatchFile(filepath.Join(tempDir, pulledManifest), string(fetched.Out.Contents()), DefaultTimeout)

			ORAS("pull", ref, "-v", "-o", pullRoot, "--include-subject").
				MatchStatus(append(foobar.FileStateKeys, foobar.AttachFileStateKey), true, 4).
				WithWorkDir(tempDir).
				WithDescription("pull attached artifact and subject").Exec()

			for _, f := range append(foobar.FileLayerNames, foobar.AttachFileName) {
				Binary("diff", filepath.Join(f), filepath.Join(pullRoot, f)).
					WithWorkDir(tempDir).
					WithDescription("download identical file " + f).Exec()
			}
		})
	})
})
