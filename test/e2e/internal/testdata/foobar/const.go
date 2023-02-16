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

package foobar

import "oras.land/oras/test/e2e/internal/utils/match"

var (
	Tag           = "foobar"
	BlobFileNames = []string{
		"foobar/foo1",
		"foobar/foo2",
		"foobar/bar",
	}
	PushFileStateKeys = []match.StateKey{
		{Digest: "2c26b46b68ff", Name: BlobFileNames[0]},
		{Digest: "2c26b46b68ff", Name: BlobFileNames[1]},
		{Digest: "fcde2b2edba5", Name: BlobFileNames[2]},
	}

	ConfigFileName     = "foobar/config.json"
	ConfigFileStateKey = match.StateKey{
		Digest: "46b68ac1696c", Name: "application/vnd.unknown.config.v1+json",
	}
	ConfigDesc = "{\"mediaType\":\"application/vnd.unknown.config.v1+json\",\"digest\":\"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a\",\"size\":2}"

	AttachFileName     = "foobar/to-be-attached"
	AttachFileMedia    = "test/oras.e2e"
	AttachFileStateKey = match.StateKey{
		Digest: "d3b29f7d12d9", Name: AttachFileName,
	}
)
