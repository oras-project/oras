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
	Digest        = "sha256:fd6ed2f36b5465244d5dc86cb4e7df0ab8a9d24adc57825099f522fe009a22bb"
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

// referrers
var (
	SignatureImageReferrerDigest = "sha256:0e007dcb9ded7f49c4dc8e3eed4a446712eb6fdf08a665a4f2352d6d2f8bdf17"
	SBOMImageReferrerDigest      = "sha256:32b78bd00723cd7d5251d4586f84d252530b7b5fe1c4104532767e6da4e04e47"
)

// fallback referrers
var (
	FallbackSignatureImageReferrerDigest = "sha256:8b3f7e000c4a6d32cd6bfcabfe874ed470d470501a09adc65afaf1c342f988ff"
	FallbackSBOMImageReferrerDigest      = "sha256:316405db72cc8f0212c19db23b498f9af8a456c9cd288f9e33acd1ba9e7cd534"
)
