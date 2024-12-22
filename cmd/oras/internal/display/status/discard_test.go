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

package status

import (
	"testing"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestDiscardHandler_OnPushSkipped(t *testing.T) {
	testDiscard := NewDiscardHandler()
	if err := testDiscard.OnManifestPushSkipped(); err != nil {
		t.Errorf("DiscardHandler.OnPushSkipped() error = %v, wantErr nil", err)
	}
}

func TestDiscardHandler_OnManifestRemoved(t *testing.T) {
	testDiscard := NewDiscardHandler()
	if err := testDiscard.OnManifestRemoved("sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a"); err != nil {
		t.Errorf("DiscardHandler.OnManifestRemoved() error = %v, wantErr nil", err)
	}
}

func TestDiscardHandler_OnIndexMerged(t *testing.T) {
	testDiscard := NewDiscardHandler()
	if err := testDiscard.OnIndexMerged("test", v1.Descriptor{}); err != nil {
		t.Errorf("DiscardHandler.OnIndexMerged() error = %v, wantErr nil", err)
	}
}

func TestDiscardHandler_OnIndexPushed(t *testing.T) {
	testDiscard := NewDiscardHandler()
	if err := testDiscard.OnIndexPushed("test"); err != nil {
		t.Errorf("DiscardHandler.OnIndexPushed() error = %v, wantErr nil", err)
	}
}
