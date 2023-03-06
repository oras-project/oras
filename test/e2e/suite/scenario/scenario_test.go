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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestORASScenarios(t *testing.T) {
	RegisterFailHandler(Fail)

	suiteConf, _ := GinkgoConfiguration()
	fmt.Printf("Starting e2e on Ginkgo node %d of total %d\n",
		suiteConf.ParallelProcess, suiteConf.ParallelTotal)
	RunSpecs(t, "ORAS Scenario Suite")
}
