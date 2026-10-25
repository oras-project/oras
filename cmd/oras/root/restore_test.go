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

package root

import (
	"context"
	"errors"
	"reflect"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/internal/display/displaytest"
	"oras.land/oras/cmd/oras/internal/display/metadata/text"
	"oras.land/oras/cmd/oras/internal/display/status"
)

func TestNewRestoreHandlers_injected(t *testing.T) {
	opts := &restoreOptions{}
	recorder := &displaytest.CallRecorder{}
	want := restoreHandlers{
		status:   &displaytest.RecordingRestoreStatusHandler{Recorder: recorder},
		metadata: &displaytest.RecordingRestoreMetadataHandler{Recorder: recorder},
	}
	got := newRestoreHandlers(opts, nil, want)
	if got.status != want.status || got.metadata != want.metadata {
		t.Fatalf("newRestoreHandlers() did not return injected handlers")
	}
}

func TestNewRestoreHandlers_default(t *testing.T) {
	opts := &restoreOptions{}
	got := newRestoreHandlers(opts, nil)
	if got.status == nil || got.metadata == nil {
		t.Fatal("newRestoreHandlers() returned nil handler")
	}
	if reflect.TypeOf(got.status) != reflect.TypeOf(status.NewTextRestoreHandler(nil, nil)) {
		t.Errorf("status handler type = %T, want text restore handler", got.status)
	}
	if reflect.TypeOf(got.metadata) != reflect.TypeOf(text.NewRestoreHandler(nil, false)) {
		t.Errorf("metadata handler type = %T, want text restore handler", got.metadata)
	}
}

func TestRecordingRestoreHandlers_callOrderAndError(t *testing.T) {
	wantedErr := errors.New("pre-copy failed")
	recorder := &displaytest.CallRecorder{}
	statusHandler := &displaytest.RecordingRestoreStatusHandler{Recorder: recorder, PreCopyErr: wantedErr}
	metadataHandler := &displaytest.RecordingRestoreMetadataHandler{Recorder: recorder}

	if err := metadataHandler.OnTagsFound([]string{"tag"}); err != nil {
		t.Fatal(err)
	}
	if err := statusHandler.PreCopy(context.Background(), ocispec.Descriptor{}); !errors.Is(err, wantedErr) {
		t.Fatalf("PreCopy() error = %v, want %v", err, wantedErr)
	}
	if got, want := recorder.Calls, []string{"OnTagsFound", "PreCopy"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("handler calls = %v, want %v", got, want)
	}
}

func TestRunRestore_injectedHandlers_reportsInputError(t *testing.T) {
	ctx := context.Background()
	recorder := &displaytest.CallRecorder{}
	opts := &restoreOptions{repository: "localhost:5000/example", input: t.TempDir() + "/missing"}
	handlers := restoreHandlers{
		status:   &displaytest.RecordingRestoreStatusHandler{Recorder: recorder},
		metadata: &displaytest.RecordingRestoreMetadataHandler{Recorder: recorder},
	}
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	opts.Remote.ApplyFlags(cmd.Flags())
	if err := runRestore(cmd, opts, handlers); err == nil {
		t.Fatal("runRestore() error = nil, want input path error")
	}
}
