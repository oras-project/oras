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
	"os"
	"testing"

	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras/cmd/oras/internal/display/status/console/testutils"
)

func TestTTYPushHandler_OnFileLoading(t *testing.T) {
	ph := NewTTYPushHandler(os.Stdout)
	if ph.OnFileLoading("test") != nil {
		t.Error("OnFileLoading() should not return an error")
	}
}

func TestTTYPushHandler_OnEmptyArtifact(t *testing.T) {
	ph := NewTTYAttachHandler(os.Stdout)
	if ph.OnEmptyArtifact() != nil {
		t.Error("OnEmptyArtifact() should not return an error")
	}
}

func TestTTYPushHandler_TrackTarget(t *testing.T) {
	// prepare pty
	_, slave, err := testutils.NewPty()
	if err != nil {
		t.Fatal(err)
	}
	defer slave.Close()
	ph := NewTTYPushHandler(slave)
	store := memory.New()
	// test
	_, err = ph.TrackTarget(store)
	if err != nil {
		t.Error("TrackTarget() should not return an error")
	}
	if ttyPushHandler, ok := ph.(*TTYPushHandler); !ok {
		t.Errorf("TrackTarget() should return a *TTYPushHandler, got %T", ttyPushHandler)
	} else if ttyPushHandler.tracked.Inner() != store {
		t.Errorf("TrackTarget() tracks unexpected tracked target: %T", ttyPushHandler.tracked)
	}
}
