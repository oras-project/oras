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
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	"oras.land/oras/test/e2e/internal/testdata/multi_arch"
	. "oras.land/oras/test/e2e/internal/utils"
	"oras.land/oras/test/e2e/internal/utils/match"
)

type resolveType uint8

const (
	resolveFrom = iota
	resolveTo
	resolveBase
)

// ResolveFlags generates resolve flags for localhost mapping.
func ResolveFlags(reg string, host string, flagType resolveType) []string {
	Expect(reg).To(HavePrefix("localhost:"), fmt.Sprintf("%q is not in format of localhost:<port>", reg))
	_, port, _ := strings.Cut(reg, ":")
	resolveFlag := "resolve"
	usernameFlag := "username"
	passwordFlag := "password"
	plainHttpFlag := "plain-http"
	fp := "--"

	switch flagType {
	case resolveFrom:
		resolveFlag = "from-" + resolveFlag
		usernameFlag = "from-" + usernameFlag
		passwordFlag = "from-" + passwordFlag
		plainHttpFlag = "from-" + plainHttpFlag
	case resolveTo:
		resolveFlag = "to-" + resolveFlag
		usernameFlag = "to-" + usernameFlag
		passwordFlag = "to-" + passwordFlag
		plainHttpFlag = "to-" + plainHttpFlag
	}

	return []string{fp + resolveFlag, fmt.Sprintf("%s:80:127.0.0.1:%s", host, port), fp + usernameFlag, Username, fp + passwordFlag, Password, fp + plainHttpFlag}
}

var _ = Describe("1.1 registry users:", func() {
	if strings.HasPrefix(Host, "localhost:") {
		When("custom host is provided", func() {
			// mockedHost represents a non-existent host name which
			// only can be resolved by custom DNS rule
			mockedHost := "oras.e2e.test"
			unary := ResolveFlags(Host, mockedHost, resolveBase)
			from := ResolveFlags(Host, mockedHost, resolveFrom)
			to := ResolveFlags(Host, mockedHost, resolveTo)
			testRepo := func(description string) string {
				return fmt.Sprintf("command/resolve/%d/%s", GinkgoRandomSeed(), description)
			}

			It("should attach a file to a subject via resolve flag", func() {
				repo := testRepo("attach")
				tempDir := PrepareTempFiles()
				prepare(RegistryRef(Host, ImageRepo, foobar.Tag), RegistryRef(Host, repo, foobar.Tag))

				ORAS(append([]string{"attach", "--artifact-type", "test/attach", RegistryRef(mockedHost, repo, foobar.Tag), fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia)}, unary...)...).
					WithWorkDir(tempDir).
					MatchStatus([]match.StateKey{foobar.AttachFileStateKey}, false, 1).Exec()
				// validate
				var index ocispec.Index
				bytes := ORAS("discover", "-o", "json", RegistryRef(Host, repo, foobar.Tag)).Exec().Out.Contents()
				Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
				Expect(len(index.Manifests)).To(Equal(1))
			})
			It("should push a blob from a stdin and output the descriptor with specific media-type", func() {
				mediaType := "test.media"
				repo := testRepo("blob/push")
				ORAS(append([]string{"blob", "push", RegistryRef(mockedHost, repo, pushDigest), "-", "--media-type", mediaType, "--descriptor", "--size", strconv.Itoa(len(pushContent))}, unary...)...).
					WithInput(strings.NewReader(pushContent)).
					MatchContent(fmt.Sprintf(pushDescFmt, mediaType)).Exec()
				ORAS("blob", "fetch", RegistryRef(Host, repo, pushDigest), "--output", "-").MatchContent(pushContent).Exec()
			})
			It("should fetch blob descriptor ", func() {
				ORAS(append([]string{"blob", "fetch", RegistryRef(mockedHost, ImageRepo, foobar.FooBlobDigest), "--descriptor"}, unary...)...).
					MatchContent(foobar.FooBlobDescriptor).Exec()
			})

			It("should delete a blob with force flag and output descriptor", func() {
				repo := testRepo("blob/delete")
				ORAS("cp", RegistryRef(Host, ImageRepo, foobar.Digest), RegistryRef(Host, repo, foobar.Digest)).Exec()
				// test
				toDeleteRef := RegistryRef(mockedHost, repo, foobar.FooBlobDigest)
				ORAS(append([]string{"blob", "delete", toDeleteRef, "--force", "--descriptor"}, unary...)...).
					MatchContent(foobar.FooBlobDescriptor).
					Exec()
				deletedRef := RegistryRef(Host, repo, foobar.FooBlobDigest)
				ORAS("blob", "delete", deletedRef).
					WithDescription("validate").
					ExpectFailure().
					MatchErrKeyWords("Error:", deletedRef, "the specified blob does not exist").
					Exec()
			})

			It("should copy an image with source resolve DNS rules", func() {
				repo := testRepo("cp/from")
				dst := RegistryRef(Host, repo, "copied")
				ORAS(append([]string{"cp", RegistryRef(mockedHost, ImageRepo, foobar.Tag), dst, "-v"}, from...)...).
					MatchStatus(foobarStates, true, len(foobarStates)).
					Exec()
				CompareRef(RegistryRef(Host, ImageRepo, foobar.Tag), dst)
			})

			It("should copy an image with destination resolve DNS rules", func() {
				src := RegistryRef(Host, ImageRepo, foobar.Tag)
				repo := testRepo("cp/to")
				tag := "copied"
				ORAS(append([]string{"cp", src, RegistryRef(mockedHost, repo, tag), "-v"}, to...)...).
					MatchStatus(foobarStates, true, len(foobarStates)).
					Exec()
				CompareRef(src, RegistryRef(Host, repo, tag))
			})

			It("should copy an image with base resolve DNS rules", func() {
				src := RegistryRef(Host, ImageRepo, foobar.Tag)
				repo := testRepo("cp/base")
				tag := "copied"
				flags := append(to[2:], "--resolve", unary[1]) //hack to chop the destination flags off
				ORAS(append([]string{"cp", src, RegistryRef(mockedHost, repo, tag), "-v"}, flags...)...).
					MatchStatus(foobarStates, true, len(foobarStates)).
					Exec()
				CompareRef(src, RegistryRef(Host, repo, tag))
			})

			It("should copy an image with overridden resolve DNS rules for both source and destination registry", func() {
				repo := testRepo("cp/override")
				tag := "copied"
				flags := append(append(to, from...), "--resolve", mockedHost+":80:1.1.1.1:5000")
				ORAS(append([]string{"cp", RegistryRef(mockedHost, ImageRepo, foobar.Tag), RegistryRef(mockedHost, repo, tag), "-v"}, flags...)...).
					MatchStatus(foobarStates, true, len(foobarStates)).
					Exec()
				CompareRef(RegistryRef(Host, ImageRepo, foobar.Tag), RegistryRef(Host, repo, tag))
			})

			It("should discover direct referrers of a subject", func() {
				bytes := ORAS(append([]string{"discover", RegistryRef(mockedHost, ArtifactRepo, foobar.Tag), "-o", "json"}, unary...)...).Exec().Out.Contents()
				var index ocispec.Index
				Expect(json.Unmarshal(bytes, &index)).ShouldNot(HaveOccurred())
				Expect(index.Manifests).To(HaveLen(2))
				Expect(index.Manifests).Should(ContainElement(foobar.SBOMImageReferrer))
				Expect(index.Manifests).Should(ContainElement(foobar.SBOMArtifactReferrer))
			})

			It("should login", func() {
				ORAS(append([]string{"login", mockedHost}, unary...)...).Exec()
			})

			It("should fetch index manifest with digest", func() {
				ORAS(append([]string{"manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.Tag)}, unary...)...).
					MatchContent(multi_arch.Manifest).Exec()
			})

			It("should fetch scratch config", func() {
				ORAS(append([]string{"manifest", "fetch-config", RegistryRef(Host, ImageRepo, foobar.Tag)}, unary...)...).
					MatchContent("{}").Exec()
			})

			It("should do forced deletion and output descriptor", func() {
				repo := testRepo("manifest/delete")
				tempTag := "to-delete"
				prepare(RegistryRef(Host, ImageRepo, foobar.Tag), RegistryRef(Host, repo, tempTag))
				ORAS(append([]string{"manifest", "delete", RegistryRef(mockedHost, repo, tempTag), "-f", "--descriptor"}, unary...)...).
					MatchContent("{\"mediaType\":\"application/vnd.oci.image.manifest.v1+json\",\"digest\":\"sha256:fd6ed2f36b5465244d5dc86cb4e7df0ab8a9d24adc57825099f522fe009a22bb\",\"size\":851}").
					WithDescription("cancel without confirmation").Exec()
				validateTag(RegistryRef(Host, repo, ""), tempTag, true)
			})

			manifest := `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"mediaType":"application/vnd.oci.image.config.v1+json","digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":53},"layers":[]}`
			digest := "sha256:a415225c47052151ea6f6f5c14750d25d580b79cc582949bb64814693365c3b4"
			It("should push a manifest from stdin without media type flag", func() {
				repo := testRepo("manifest/push")
				tag := "from-stdin"
				prepare(RegistryRef(Host, ImageRepo, foobar.Digest), RegistryRef(Host, repo, foobar.Digest))
				//test
				ORAS(append([]string{"manifest", "push", RegistryRef(Host, repo, tag), "-"}, unary...)...).
					MatchKeyWords("Pushed", RegistryRef(Host, repo, tag), "Digest:", digest).
					WithInput(strings.NewReader(manifest)).Exec()
				validateTag(RegistryRef(Host, repo, ""), tag, false)
			})

			It("should pull all files in an image", func() {
				pullRoot := "pulled"
				configName := "test.config"
				tempDir := PrepareTempFiles()
				stateKeys := append(foobar.ImageLayerStateKeys, foobar.ManifestStateKey, foobar.ImageConfigStateKey(configName))
				// test
				ORAS(append([]string{"pull", RegistryRef(mockedHost, ImageRepo, foobar.Tag), "-v", "--config", configName, "-o", pullRoot}, unary...)...).
					MatchStatus(stateKeys, true, len(stateKeys)).
					WithWorkDir(tempDir).Exec()
				// validate config and layers
				configPath := filepath.Join(tempDir, pullRoot, configName)
				Expect(configPath).Should(BeAnExistingFile())
				f, err := os.Open(configPath)
				Expect(err).ShouldNot(HaveOccurred())
				defer f.Close()
				Eventually(gbytes.BufferReader(f)).Should(gbytes.Say("{}"))
				for _, f := range foobar.ImageLayerNames {
					Binary("diff", filepath.Join(tempDir, "foobar", f), filepath.Join(pullRoot, f)).
						WithWorkDir(tempDir).
						Exec()
				}
			})

			It("should push files without customized media types", func() {
				repo := testRepo("push")
				tempDir := PrepareTempFiles()
				ORAS(append([]string{"push", RegistryRef(mockedHost, repo, foobar.Tag), "-v"}, append(foobar.FileLayerNames, unary...)...)...).
					MatchStatus(foobar.FileStateKeys, true, len(foobar.FileStateKeys)).
					WithWorkDir(tempDir).Exec()
			})

			It("should list repositories", func() {
				ORAS(append([]string{"repository", "list", mockedHost}, unary...)...).MatchKeyWords(ImageRepo).Exec()
			})

			It("should list tags", func() {
				ORAS(append([]string{"repository", "show-tags", RegistryRef(mockedHost, ImageRepo, foobar.Digest)}, unary...)...).MatchKeyWords(foobar.Tag).Exec()
			})

			It("should add a tag to an existent manifest when providing tag reference", func() {
				repo := testRepo("tag")
				prepare(RegistryRef(Host, ImageRepo, foobar.Digest), RegistryRef(Host, repo, foobar.Digest))
				newTag := "tag-via-digest"
				ORAS(append([]string{"tag", RegistryRef(mockedHost, repo, foobar.Digest), newTag}, unary...)...).MatchKeyWords(newTag).Exec()
				ORAS("repo", "tags", RegistryRef(Host, repo, "")).MatchKeyWords(newTag).Exec()
			})
		})
	}
})
