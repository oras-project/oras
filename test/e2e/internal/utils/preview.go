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

	. "github.com/onsi/ginkgo/v2"
)

func RunAndShowPreviewInHelp(args []string, keywords ...string) {
	It(fmt.Sprintf("should run %q command", strings.Join(args, " ")), func() {
		ORAS(append(args, "--help")...).
			MatchKeyWords(append(keywords, "[Preview] "+args[len(args)-1], "\nUsage:")...).
			WithDescription("show preview and help doc").
			Exec()
	})
}
