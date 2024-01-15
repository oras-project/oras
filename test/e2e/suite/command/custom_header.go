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
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	"oras.land/oras/test/e2e/internal/testdata/foobar"
	"oras.land/oras/test/e2e/internal/testdata/multi_arch"
	. "oras.land/oras/test/e2e/internal/utils"
)

var _ = Describe("1.1 registry users:", func() {
	headerTestRepo := func(text string) string {
		return fmt.Sprintf("command/headertest/%d/%s", GinkgoRandomSeed(), text)
	}
	var (
		FoobarHeaderInput = "Foo:bar"
		FoobarHeader      = "\"Foo\": \"bar\"\n"
		AbHeaderInput     = "A: b"
		AbHeader          = "\"A\": \" b\"\n"
	)
	When("custom header is provided", func() {
		It("attach", func() {
			testRepo := headerTestRepo("attach")
			tempDir := PrepareTempFiles()
			subjectRef := RegistryRef(Host, testRepo, foobar.Tag)
			prepare(RegistryRef(Host, ImageRepo, foobar.Tag), subjectRef)
			ORAS("attach", "--artifact-type", "test/attach", subjectRef,
				fmt.Sprintf("%s:%s", foobar.AttachFileName, foobar.AttachFileMedia),
				"-d", "-H", FoobarHeaderInput, "-H", AbHeaderInput).
				WithWorkDir(tempDir).MatchRequestHeaders(FoobarHeader, AbHeader).Exec()
		})
		It("blob", func() {
			blobDigest := "sha256:2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae"
			ORAS("blob", "fetch", RegistryRef(Host, ImageRepo, blobDigest), "--descriptor",
				"-d", "-H", FoobarHeaderInput, "-H", AbHeaderInput).
				MatchRequestHeaders(FoobarHeader, AbHeader).Exec()
		})
		It("manifest", func() {
			ORAS("manifest", "fetch", RegistryRef(Host, ImageRepo, multi_arch.Tag),
				"-d", "-H", FoobarHeaderInput, "-H", AbHeaderInput).
				MatchRequestHeaders(FoobarHeader, AbHeader).Exec()
		})
		It("pull", func() {
			tempDir := GinkgoT().TempDir()
			ORAS("pull", "-d", "-H", FoobarHeaderInput, "-H", AbHeaderInput,
				RegistryRef(Host, ImageRepo, "foobar"), "--config", "config.json").
				WithWorkDir(tempDir).MatchRequestHeaders(FoobarHeader, AbHeader).Exec()
		})
		It("push", func() {
			repo := headerTestRepo("push")
			tempDir := PrepareTempFiles()
			ORAS("push", "-d", "-H", FoobarHeaderInput, "-H", AbHeaderInput,
				RegistryRef(Host, repo, "latest"), "foobar/bar").
				WithWorkDir(tempDir).MatchRequestHeaders(FoobarHeader, AbHeader).Exec()
		})
		It("repo", func() {
			ORAS("repository", "list", Host, "-d", "-H", FoobarHeaderInput, "-H", AbHeaderInput).
				MatchRequestHeaders(FoobarHeader, AbHeader).Exec()
		})
		It("tag", func() {
			digest := "sha256:e2bfc9cc6a84ec2d7365b5a28c6bc5806b7fa581c9ad7883be955a64e3cc034f"
			ORAS("tag", RegistryRef(Host, ImageRepo, digest), "latest",
				"-d", "-H", FoobarHeaderInput, "-H", AbHeaderInput).
				MatchRequestHeaders(FoobarHeader, AbHeader).Exec()
		})
		It("login", func() {
			ORAS("login", Host, "-u", Username, "-p", Password, "--registry-config", filepath.Join(GinkgoT().TempDir(), "test.config"),
				"-H", FoobarHeaderInput, "-H", AbHeaderInput).
				MatchRequestHeaders(FoobarHeader, AbHeader).Exec()
		})
		It("copy and add custom headers in source registry requests", func() {
			headerTestRepo := headerTestRepo("from-header")
			src := RegistryRef(Host, ImageRepo, foobar.Tag)
			dst := RegistryRef(Host, headerTestRepo, "fromHeader")
			ORAS("cp", src, dst, "-d", "--from-header", FoobarHeaderInput, "--from-header", AbHeaderInput).
				MatchCpRequestHeaders(Host, ImageRepo, FoobarHeader, AbHeader).Exec()
		})
		It("copy and add custom headers in destination registry requests", func() {
			headerTestRepo := headerTestRepo("to-header")
			src := RegistryRef(Host, ImageRepo, foobar.Tag)
			dst := RegistryRef(Host, headerTestRepo, "toHeader")
			ORAS("cp", src, dst, "-d", "--to-header", FoobarHeaderInput, "--to-header", AbHeaderInput).
				MatchCpRequestHeaders(Host, headerTestRepo, FoobarHeader, AbHeader).Exec()
		})
	})
})

var _ = Describe("OCI image layout users:", func() {
	When("custom header is provided", func() {
		It("should fail attach", func() {
			ORAS("attach", ".:test", "-a", "test=true", "--artifact-type", "doc/example", "--oci-layout", "-H=foo:bar").
				WithWorkDir(GinkgoT().TempDir()).
				ExpectFailure().
				MatchErrKeyWords("custom header").
				Exec()
		})
	})
})
