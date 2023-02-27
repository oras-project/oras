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
	"bytes"
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	. "oras.land/oras/test/e2e/internal/utils"
	"oras.land/oras/test/e2e/internal/utils/match"
)

var _ = Describe("Remote registry users:", func() {
	layerDescriptorTemplate := `{"mediaType":"%s","digest":"sha256:fcde2b2edba56bf408601fb721fe9b5c338d10ee429ea04fae5511b68fbf8fb9","size":3,"annotations":{"org.opencontainers.image.title":"foobar/bar"}}`
	tag := "e2e"
	When("pushing to registy without OCI artifact support", func() {
		repoPrefix := fmt.Sprintf("command/push/%d", GinkgoRandomSeed())
		files := []string{
			"foobar/config.json",
			"foobar/bar",
		}
		statusKeys := []match.StateKey{
			{Digest: "44136fa355b3", Name: "application/vnd.unknown.config.v1+json"},
			{Digest: "fcde2b2edba5", Name: files[1]},
		}
		configDescriptorTemplate := `{"mediaType":"%s","digest":"sha256:46b68ac1696c3870d537f376868d9402400de28587e345264a77b65da09669be","size":13}`

		It("should push files without customized media types", func() {
			repo := fmt.Sprintf("%s/%s", repoPrefix, "with-mediatype")
			tempDir := GinkgoT().TempDir()
			if err := CopyTestData(tempDir); err != nil {
				panic(err)
			}

			ORAS("push", Reference(Host, repo, tag), files[1], "-v").
				MatchStatus(statusKeys, true, len(statusKeys)).
				WithWorkDir(tempDir).Exec()
			fetched := ORAS("manifest", "fetch", Reference(Host, repo, tag)).Exec().Out
			Binary("jq", ".blobs[]", "--compact-output").
				MatchTrimmedContent(fmt.Sprintf(layerDescriptorTemplate, ocispec.MediaTypeImageLayer)).
				WithInput(fetched).Exec()
		})

		It("should push files and tag", func() {
			repo := fmt.Sprintf("%s/%s", repoPrefix, "multi-tag")
			tempDir := CopyTestDataToTemp()
			extraTag := "2e2"

			ORAS("push", fmt.Sprintf("%s,%s", Reference(Host, repo, tag), extraTag), files[1], "-v").
				MatchStatus(statusKeys, true, 1).
				WithWorkDir(tempDir).Exec()
			fetched := ORAS("manifest", "fetch", Reference(Host, repo, tag)).Exec().Out
			Binary("jq", ".blobs[]", "--compact-output").
				MatchTrimmedContent(fmt.Sprintf(layerDescriptorTemplate, ocispec.MediaTypeImageLayer)).
				WithInput(fetched).Exec()

			fetched = ORAS("manifest", "fetch", Reference(Host, repo, extraTag)).Exec().Out
			Binary("jq", ".blobs[]", "--compact-output").
				MatchTrimmedContent(fmt.Sprintf(layerDescriptorTemplate, ocispec.MediaTypeImageLayer)).
				WithInput(fetched).Exec()
		})

		It("should push files with customized media types", func() {
			repo := fmt.Sprintf("%s/%s", repoPrefix, "layer-mediatype")
			layerType := "layer.type"
			tempDir := GinkgoT().TempDir()
			if err := CopyTestData(tempDir); err != nil {
				panic(err)
			}
			ORAS("push", Reference(Host, repo, tag), files[1]+":"+layerType, "-v").
				MatchStatus(statusKeys, true, 1).
				WithWorkDir(tempDir).Exec()
			fetched := ORAS("manifest", "fetch", Reference(Host, repo, tag)).Exec().Out
			Binary("jq", ".blobs[]", "--compact-output").
				MatchTrimmedContent(fmt.Sprintf(layerDescriptorTemplate, layerType)).
				WithInput(fetched).Exec()
		})

		It("should push files with manifest exported", func() {
			repo := fmt.Sprintf("%s/%s", repoPrefix, "export-manifest")
			layerType := "layer.type"
			tempDir := GinkgoT().TempDir()
			if err := CopyTestData(tempDir); err != nil {
				panic(err)
			}

			exportPath := "packed.json"
			ORAS("push", Reference(Host, repo, tag), files[1]+":"+layerType, "-v", "--export-manifest", exportPath).
				MatchStatus(statusKeys, true, len(statusKeys)).
				WithWorkDir(tempDir).Exec()
			fetched := ORAS("manifest", "fetch", Reference(Host, repo, tag)).Exec().Out.Contents()
			MatchFile(filepath.Join(tempDir, exportPath), string(fetched), DefaultTimeout)
		})

		It("should push files with customized config file", func() {
			repo := fmt.Sprintf("%s/%s", repoPrefix, "config")
			tempDir := GinkgoT().TempDir()
			if err := CopyTestData(tempDir); err != nil {
				panic(err)
			}

			ORAS("push", Reference(Host, repo, tag), "--config", files[0], files[1], "-v").
				MatchStatus([]match.StateKey{
					{Digest: "46b68ac1696c", Name: oras.MediaTypeUnknownConfig},
					{Digest: "fcde2b2edba5", Name: files[1]},
				}, true, 2).
				WithWorkDir(tempDir).Exec()
			fetched := ORAS("manifest", "fetch", Reference(Host, repo, tag)).Exec().Out
			Binary("jq", ".config", "--compact-output").
				MatchTrimmedContent(fmt.Sprintf(configDescriptorTemplate, oras.MediaTypeUnknownConfig)).
				WithInput(fetched).Exec()
		})

		It("should push files with customized config file and mediatype", func() {
			repo := fmt.Sprintf("%s/%s", repoPrefix, "config-mediatype")
			configType := "config.type"
			tempDir := GinkgoT().TempDir()
			if err := CopyTestData(tempDir); err != nil {
				panic(err)
			}

			ORAS("push", Reference(Host, repo, tag), "--config", fmt.Sprintf("%s:%s", files[0], configType), files[1], "-v").
				MatchStatus([]match.StateKey{
					{Digest: "46b68ac1696c", Name: configType},
					{Digest: "fcde2b2edba5", Name: files[1]},
				}, true, 2).
				WithWorkDir(tempDir).Exec()
			fetched := ORAS("manifest", "fetch", Reference(Host, repo, tag)).Exec().Out
			Binary("jq", ".config", "--compact-output").
				MatchTrimmedContent(fmt.Sprintf(configDescriptorTemplate, configType)).
				WithInput(fetched).Exec()
		})

		It("should push files with customized manifest annotation", func() {
			repo := fmt.Sprintf("%s/%s", repoPrefix, "manifest-annotation")
			key := "image-anno-key"
			value := "image-anno-value"
			tempDir := GinkgoT().TempDir()
			if err := CopyTestData(tempDir); err != nil {
				panic(err)
			}

			ORAS("push", Reference(Host, repo, tag), files[1], "-v", "--annotation", fmt.Sprintf("%s=%s", key, value)).
				MatchStatus(statusKeys, true, len(statusKeys)).
				WithWorkDir(tempDir).Exec()
			fetched := ORAS("manifest", "fetch", Reference(Host, repo, tag)).Exec().Out

			Binary("jq", `.annotations|del(.["org.opencontainers.artifact.created"])`, "--compact-output").
				MatchTrimmedContent(fmt.Sprintf(`{"%s":"%s"}`, key, value)).
				WithInput(fetched).Exec()
		})

		It("should push files with customized file annotation", func() {
			repo := fmt.Sprintf("%s/%s", repoPrefix, "file-annotation")
			tempDir := GinkgoT().TempDir()
			if err := CopyTestData(tempDir); err != nil {
				panic(err)
			}

			ORAS("push", Reference(Host, repo, tag), files[1], "-v", "--annotation-file", "foobar/annotation.json", "--config", files[0]).
				MatchStatus(statusKeys, true, 1).
				WithWorkDir(tempDir).Exec()
			fetched := ORAS("manifest", "fetch", Reference(Host, repo, tag)).Exec().Out

			// see testdata\files\foobar\annotation.json
			Binary("jq", `.annotations|del(.["org.opencontainers.image.created"])`, "--compact-output").
				MatchTrimmedContent(fmt.Sprintf(`{"%s":"%s"}`, "hi", "manifest")).
				WithInput(bytes.NewReader(fetched.Contents())).Exec()

			Binary("jq", ".config.annotations", "--compact-output").
				MatchTrimmedContent(fmt.Sprintf(`{"%s":"%s"}`, "hello", "config")).
				WithInput(bytes.NewReader(fetched.Contents())).Exec()

			Binary("jq", `.layers[0].annotations|del(.["org.opencontainers.image.title"])`, "--compact-output").
				MatchTrimmedContent(fmt.Sprintf(`{"%s":"%s"}`, "foo", "bar")).
				WithInput(bytes.NewReader(fetched.Contents())).Exec()
		})
	})
})
