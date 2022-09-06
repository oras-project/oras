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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	"oras.land/oras/test/e2e/utils"
	"oras.land/oras/test/e2e/utils/match"
)

var _ = Context("ORAS user", Ordered, func() {
	repo := "oras-artifact"
	Describe("logs in", func() {
		When("should succeed with basic auth", func() {
			utils.Exec(match.NewOption(strings.NewReader(PASSWORD), match.Content("Login Succeeded\n"), nil, false),
				"should succeed with username flag and password from stdin",
				"login", utils.Host, "-u", USERNAME, "--password-stdin")
		})
	})

	Describe("pushes image and check", Ordered, func() {
		tag := "image"
		When("pushing an image", func() {
			status := match.NewStatus([]match.StateKey{
				{Digest: "b5bb9d8014a0", Name: "foo1"},
				{Digest: "b5bb9d8014a0", Name: "foo2"},
				{Digest: "7d865e959b24", Name: "bar"},
				{Digest: "e3b0c44298fc", Name: "application/vnd.unknown.config.v1+json"},
				{Digest: "992db6dcc803", Name: "application/vnd.oci.image.manifest.v1+json"},
			}, *match.MatchableStatus("push", true))
			utils.Exec(match.NewOption(nil, status, nil, false), "should succeed with username flag and password from stdin",
				"push", utils.Reference(utils.Host, repo, tag), utils.ImageBlob("foobar/foo1"), utils.ImageBlob("foobar/foo2"), utils.ImageBlob("foobar/bar"), "-v")
		})
	})
})
