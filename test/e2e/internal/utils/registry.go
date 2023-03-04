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

package utils

import (
	"fmt"
	"strings"

	"github.com/onsi/gomega"
	"oras.land/oras-go/v2/registry"
)

// Reference generates the reference string from given parameters.
func Reference(reg string, repo string, tagOrDigest string) string {
	ref := registry.Reference{
		Registry:   reg,
		Repository: repo,
		Reference:  strings.TrimSpace(tagOrDigest),
	}
	gomega.Expect(ref.Validate()).ShouldNot(gomega.HaveOccurred())
	return ref.String()
}

type resolveType uint8

const (
	Base resolveType = iota
	From
	To
)

// ResolveFlags generates resolve flags for localhost mapping.
func ResolveFlags(reg string, host string, flagType resolveType) []string {
	gomega.Expect(reg).To(gomega.HavePrefix("localhost:"), fmt.Sprintf("%q is not in format of localhost:<port>", reg))
	_, port, _ := strings.Cut(reg, ":")
	resolveFlag := "resolve"
	usernameFlag := "username"
	passwordFlag := "password"
	plainHttpFlag := "plain-http"
	fp := "--"

	switch flagType {
	case From:
		resolveFlag = "from-" + resolveFlag
		usernameFlag = "from-" + usernameFlag
		passwordFlag = "from-" + passwordFlag
		plainHttpFlag = "from-" + plainHttpFlag
	case To:
		resolveFlag = "to-" + resolveFlag
		usernameFlag = "to-" + usernameFlag
		passwordFlag = "to-" + passwordFlag
		plainHttpFlag = "to-" + plainHttpFlag
	}

	return []string{fp + resolveFlag, fmt.Sprintf("%s:80:127.0.0.1:%s", host, port), fp + usernameFlag, Username, fp + passwordFlag, Password, fp + plainHttpFlag}
}
