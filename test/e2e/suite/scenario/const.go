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

import "oras.land/oras/test/e2e/internal/utils/match"

var (
	blobFileNames = []string{
		"foobar/foo1",
		"foobar/foo2",
		"foobar/bar",
	}
	pushFileStateKeys = []match.StateKey{
		{Digest: "2c26b46b68ff", Name: blobFileNames[0]},
		{Digest: "2c26b46b68ff", Name: blobFileNames[1]},
		{Digest: "fcde2b2edba5", Name: blobFileNames[2]},
	}

	configFileName     = "foobar/config.json"
	configFileStateKey = match.StateKey{
		Digest: "46b68ac1696c", Name: "application/vnd.unknown.config.v1+json",
	}

	attachFileName     = "foobar/to-be-attached"
	attachFileMedia    = "test/oras.e2e"
	attachFileStateKey = match.StateKey{
		Digest: "d3b29f7d12d9", Name: attachFileName,
	}
)
