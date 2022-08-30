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
	. "github.com/onsi/ginkgo/v2"
	"oras.land/oras/test/e2e/test"
)

const (
	USERNAME = "hello"
	PASSWORD = "oras-test"
)

var _ = Context("ORAS user", Ordered, func() {
	Describe("runs commands without login", func() {
		When("running attach command", func() {
			test.ExecAndMatchErrKeyWords("should fail and show error",
				[]string{"attach", test.Host + "/hello:test", "hi.txt", "--artifact-type", "doc/example"},
				[]string{"Error:", "credential required"},
			)
		})
	})
})
