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

package progress

import (
	"regexp"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func Test_status_Render(t *testing.T) {
	escRegexp := regexp.MustCompile("\x1b\\[[0-9]+m")
	equal := func(got, want [2]string) bool {
		noColor := [2]string{
			escRegexp.ReplaceAllString(got[0], ""),
			got[1],
		}
		return noColor == want
	}
	tests := []struct {
		name  string
		s     *status
		width int
		want  [2]string
	}{
		{
			name: "default status",
			s: newStatus(ocispec.Descriptor{
				MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
				Size:      1234567890,
				Digest:    "sha256:c775e7b757ede630cd0aa1113bd102661ab38829ca52a6422ab782862f268646",
				Annotations: map[string]string{
					"org.opencontainers.image.title": "hello.bin",
				},
			}),
			width: 80,
			want: [2]string{
				"⠋ [....................](   0  B/s)  hello.bin          -/1.15 GB   0.00%     0s",
				"  └─ sha256:c775e7b757ede630cd0aa1113bd102661ab38829ca52a6422ab782862f268646    ",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.Render(tt.width); !equal(got, tt.want) {
				t.Errorf("status.Render() = %v, want %v", got, tt.want)
			}
		})
	}
}
