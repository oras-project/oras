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

package metadata

import (
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestDiscard_OnTagged(t *testing.T) {
	testDiscard := NewDiscardHandler()
	if err := testDiscard.OnTagged(ocispec.Descriptor{}, "test"); err != nil {
		t.Errorf("testDiscard.OnTagged() error = %v, want nil", err)
	}
}

func TestDiscardHandler_OnManifestPushed(t *testing.T) {
	testDiscard := NewDiscardHandler()
	if err := testDiscard.OnManifestPushed(ocispec.Descriptor{}); err != nil {
		t.Errorf("DiscardHandler.OnManifestPushed() error = %v, wantErr nil", err)
	}
}
