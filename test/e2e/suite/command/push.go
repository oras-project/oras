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
	"sync"

	. "github.com/onsi/ginkgo/v2"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	. "oras.land/oras/test/e2e/internal/utils"
	"oras.land/oras/test/e2e/internal/utils/match"
)

var _ = Describe("Remote registry users:", func() {
	files := []string{
		"foobar/config.json",
		"foobar/bar",
	}
	statusKeys := []match.StateKey{
		{Digest: "44136fa355b3", Name: "application/vnd.unknown.config.v1+json"},
		{Digest: "fcde2b2edba5", Name: files[1]},
	}

	layerDescriptorTemplate := `{"mediaType":"%s","digest":"sha256:fcde2b2edba56bf408601fb721fe9b5c338d10ee429ea04fae5511b68fbf8fb9","size":3,"annotations":{"org.opencontainers.image.title":"foobar/bar"}}`
	configDescriptorTemplate := `{"mediaType":"%s","digest":"sha256:46b68ac1696c3870d537f376868d9402400de28587e345264a77b65da09669be","size":13}`
	repo := "command/push"
	var tempDir string
	var lock sync.Mutex
	BeforeEach(func() {
		if tempDir != "" {
			return
		}
		lock.Lock()
		defer lock.Unlock()
		if tempDir != "" {
			return
		}
		tempDir = GinkgoT().TempDir()
		fmt.Printf("Prepared temporary working directory in %s", tempDir)
		if err := CopyTestData(tempDir); err != nil {
			panic(err)
		}
	})

	When("pushing to registy without OCI artifact support", func() {
		It("should push files without customized media types", func() {
			tag := "no-mediatype"
			ORAS("push", Reference(Host, repo, tag), files[1], "-v").
				MatchStatus(statusKeys, true, 2).
				WithWorkDir(tempDir).Exec()
			fetched := ORAS("manifest", "fetch", Reference(Host, repo, tag)).Exec().Out
			Binary("jq", ".layers[]", "--compact-output").
				MatchContent(fmt.Sprintf(layerDescriptorTemplate, ocispec.MediaTypeImageLayer)).
				WithInput(fetched).Exec()
		})

		It("should push files with customized media types", func() {
			tag := "layer-mediatype"
			layerType := "layer.type"
			ORAS("push", Reference(Host, repo, tag), files[1]+":"+layerType, "-v").
				MatchStatus(statusKeys, true, 2).
				WithWorkDir(tempDir).Exec()
			fetched := ORAS("manifest", "fetch", Reference(Host, repo, tag)).Exec().Out
			Binary("jq", ".layers[]", "--compact-output").
				MatchContent(fmt.Sprintf(layerDescriptorTemplate, layerType)).
				WithInput(fetched).Exec()
		})

		It("should push files with manifest exported", func() {
			tag := "exported"
			layerType := "layer.type"
			exportPath := "packed.json"
			ORAS("push", Reference(Host, repo, tag), files[1]+":"+layerType, "-v", "--export-manifest", exportPath).
				MatchStatus(statusKeys, true, 2).
				WithWorkDir(tempDir).Exec()
			fetched := ORAS("manifest", "fetch", Reference(Host, repo, tag)).Exec().Out
			Binary("cat", exportPath).
				WithWorkDir(tempDir).
				MatchContent(string(fetched.Contents())).Exec()
		})

		It("should push files with customized config file", func() {
			tag := "config"
			ORAS("push", Reference(Host, repo, tag), "--config", files[0], files[1], "-v").
				MatchStatus(statusKeys, true, 2).
				WithWorkDir(tempDir).Exec()
			fetched := ORAS("manifest", "fetch", Reference(Host, repo, tag)).Exec().Out
			Binary("jq", ".config", "--compact-output").
				MatchContent(fmt.Sprintf(configDescriptorTemplate, oras.MediaTypeUnknownConfig)).
				WithInput(fetched).Exec()
		})

		It("should push files with customized config file and mediatype", func() {
			tag := "config-mediatype"
			configType := "config.type"
			ORAS("push", Reference(Host, repo, tag), "--config", fmt.Sprintf("%s:%s", files[0], configType), files[1], "-v").
				MatchStatus([]match.StateKey{
					{Digest: "46b68ac1696c", Name: configType},
					{Digest: "fcde2b2edba5", Name: files[1]},
				}, true, 2).
				WithWorkDir(tempDir).Exec()
			fetched := ORAS("manifest", "fetch", Reference(Host, repo, tag)).Exec().Out
			Binary("jq", ".config", "--compact-output").
				MatchContent(fmt.Sprintf(configDescriptorTemplate, configType)).
				WithInput(fetched).Exec()
		})
	})
})
