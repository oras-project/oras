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

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"oras.land/oras/test/e2e/test"
)

const (
	digest_latest         = "sha256:7d246653d0511db2a6b2e0436cfd0e52ac8c066000264b3ce63331ac66dca625"
	manifest_latest       = `{"manifests":[{"digest":"sha256:f54a58bc1aac5ea1a25d796ae155dc228b3f0e11d046ae276b39c4bf2f13d8c4","mediaType":"application\/vnd.docker.distribution.manifest.v2+json","platform":{"architecture":"amd64","os":"linux"},"size":525},{"digest":"sha256:7b8b7289d0536a08eabdf71c20246e23f7116641db7e1d278592236ea4dcb30c","mediaType":"application\/vnd.docker.distribution.manifest.v2+json","platform":{"architecture":"arm","os":"linux","variant":"v5"},"size":525},{"digest":"sha256:f130bd2d67e6e9280ac6d0a6c83857bfaf70234e8ef4236876eccfbd30973b1c","mediaType":"application\/vnd.docker.distribution.manifest.v2+json","platform":{"architecture":"arm","os":"linux","variant":"v7"},"size":525},{"digest":"sha256:432f982638b3aefab73cc58ab28f5c16e96fdb504e8c134fc58dff4bae8bf338","mediaType":"application\/vnd.docker.distribution.manifest.v2+json","platform":{"architecture":"arm64","os":"linux","variant":"v8"},"size":525},{"digest":"sha256:995efde2e81b21d1ea7066aa77a59298a62a9e9fbb4b77f36c189774ec9b1089","mediaType":"application\/vnd.docker.distribution.manifest.v2+json","platform":{"architecture":"386","os":"linux"},"size":525},{"digest":"sha256:eb11b1a194ff8e236a01eff392c4e1296a53b0fb4780d8b0382f7996a15d5392","mediaType":"application\/vnd.docker.distribution.manifest.v2+json","platform":{"architecture":"mips64le","os":"linux"},"size":525},{"digest":"sha256:b836bb24a270b9cc935962d8228517fde0f16990e88893d935efcb1b14c0017a","mediaType":"application\/vnd.docker.distribution.manifest.v2+json","platform":{"architecture":"ppc64le","os":"linux"},"size":525},{"digest":"sha256:98c9722322be649df94780d3fbe594fce7996234b259f27eac9428b84050c849","mediaType":"application\/vnd.docker.distribution.manifest.v2+json","platform":{"architecture":"riscv64","os":"linux"},"size":525},{"digest":"sha256:c7b6944911848ce39b44ed660d95fb54d69bbd531de724c7ce6fc9f743c0b861","mediaType":"application\/vnd.docker.distribution.manifest.v2+json","platform":{"architecture":"s390x","os":"linux"},"size":525},{"digest":"sha256:3624dfaed3b147d49409b0306a2faedfc8da7117b1b59d81714632cef2367e57","mediaType":"application\/vnd.docker.distribution.manifest.v2+json","platform":{"architecture":"amd64","os":"windows","os.version":"10.0.20348.887"},"size":1125},{"digest":"sha256:f220cf100ada1cad5d2c1ce8aa6765da9a261f4cb3911ba5a1bf039769fa117b","mediaType":"application\/vnd.docker.distribution.manifest.v2+json","platform":{"architecture":"amd64","os":"windows","os.version":"10.0.17763.3287"},"size":1125}],"mediaType":"application\/vnd.docker.distribution.manifest.list.v2+json","schemaVersion":2}`
	descriptor_latest     = `{"mediaType":"application/vnd.docker.distribution.manifest.list.v2+json","digest":"sha256:7d246653d0511db2a6b2e0436cfd0e52ac8c066000264b3ce63331ac66dca625","size":2562}`
	descriptor_linuxAMD64 = `{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","digest":"sha256:f54a58bc1aac5ea1a25d796ae155dc228b3f0e11d046ae276b39c4bf2f13d8c4","size":525}`
	digest_linuxAMD64     = "sha256:f54a58bc1aac5ea1a25d796ae155dc228b3f0e11d046ae276b39c4bf2f13d8c4"
	manifest_linuxAMD64   = `{
   "schemaVersion": 2,
   "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
   "config": {
      "mediaType": "application/vnd.docker.container.image.v1+json",
      "size": 1469,
      "digest": "sha256:feb5d9fea6a5e9606aa995e879d862b825965ba48de054caab5ef356dc6b3412"
   },
   "layers": [
      {
         "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
         "size": 2479,
         "digest": "sha256:2db29710123e3e53a794f2694094b9b4338aa9ee5c40b930cb8063a1be392c54"
      }
   ]
}`
)

var orasPath string
var _ = BeforeSuite(func() {
	orasPath = os.Getenv("ORAS_PATH")

	if orasPath == "" {
		// fallback to native build to facilitate locally debugging
		var err error
		orasPath, err = gexec.Build("oras.land/oras/cmd/oras")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		DeferCleanup(gexec.CleanupBuildArtifacts)
	}
})

var _ = Context("ORAS", func() {
	Describe("runs manifest command", func() {
		When("looking for supported command and help", func() {
			test.ExecAndMatchOutKeyWords("should show available commands",
				&orasPath,
				[]string{"manifest", "-h"},
				[]string{"[Preview] Manifest", "[Preview] Fetch"})

			test.ExecAndMatchOutKeyWords("should show fetch-related help doc",
				&orasPath,
				[]string{"manifest", "fetch", "-h"},
				[]string{"** This command is in preview and under development. **", "[Preview] Fetch"})

			test.ExecAndMatchOutKeyWords("should also work with alias get",
				&orasPath,
				[]string{"manifest", "fetch", "-h"},
				[]string{"** This command is in preview and under development. **", "[Preview] Fetch"})
		})

		When("fetching manifest with no artifact reference provided", func() {
			test.ExecAndMatchErrKeyWords("should fail",
				&orasPath,
				[]string{"manifest", "fetch"},
				[]string{"Error:"})
		})

		When("fetching manifest list content", func() {
			test.ExecAndMatchOut("should fetch manifest list with tag",
				&orasPath,
				[]string{"manifest", "fetch", "docker.io/library/hello-world:latest"},
				manifest_latest)

			test.ExecAndMatchOut("should fetch manifest list with digest",
				&orasPath,
				[]string{"manifest", "fetch", "docker.io/library/hello-world@" + digest_latest},
				manifest_latest)
		})

		When("fetching manifest content", func() {
			test.ExecAndMatchOut("should fetch manifest with target platform",
				&orasPath,
				[]string{"manifest", "fetch", "docker.io/library/hello-world@" + digest_latest, "--platform", "linux/amd64"},
				manifest_linuxAMD64)

			test.ExecAndMatchOut("should fetch manifest with platform validation",
				&orasPath,
				[]string{"manifest", "fetch", "docker.io/library/hello-world@" + digest_linuxAMD64, "--platform", "linux/amd64"},
				manifest_linuxAMD64)
		})

		When("fetching descriptor", func() {
			test.ExecAndMatchOut("should fetch descriptor with digest",
				&orasPath,
				[]string{"manifest", "fetch", "docker.io/library/hello-world@" + digest_latest, "--descriptor"},
				descriptor_latest)

			test.ExecAndMatchOut("should fetch descriptor with target platform",
				&orasPath,
				[]string{"manifest", "fetch", "docker.io/library/hello-world@" + digest_latest, "--platform", "linux/amd64", "--descriptor"},
				descriptor_linuxAMD64)

			test.ExecAndMatchOut("should fetch manifest with platform validation",
				&orasPath,
				[]string{"manifest", "fetch", "docker.io/library/hello-world@" + digest_linuxAMD64, "--platform", "linux/amd64", "--descriptor"},
				descriptor_linuxAMD64)
		})
	})
})
