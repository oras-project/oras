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

package text

import (
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
)

func TestAttachHandler_OnAttached(t *testing.T) {
	type fields struct {
		printer            *output.Printer
		subjectRefByDigest string
		root               ocispec.Descriptor
	}
	type args struct {
		target  *option.Target
		root    ocispec.Descriptor
		subject ocispec.Descriptor
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ah := &AttachHandler{
				printer:            tt.fields.printer,
				subjectRefByDigest: tt.fields.subjectRefByDigest,
				root:               tt.fields.root,
			}
			ah.OnAttached(tt.args.target, tt.args.root, tt.args.subject)
		})
	}
}
