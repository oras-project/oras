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

package repo

import (
	"errors"
	"reflect"
	"testing"

	"oras.land/oras/cmd/oras/internal/display/metadata/text"
	"oras.land/oras/cmd/oras/internal/option"
)

type recordingRepoTagsHandler struct {
	calls []string
	err   error
}

func (h *recordingRepoTagsHandler) OnTagListed(string) error {
	h.calls = append(h.calls, "OnTagListed")
	return h.err
}

func (h *recordingRepoTagsHandler) Render() error {
	h.calls = append(h.calls, "Render")
	return nil
}

func TestNewRepoTagsHandler_injected(t *testing.T) {
	want := &recordingRepoTagsHandler{}
	got, err := newRepoTagsHandler(&showTagsOptions{}, want)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatal("newRepoTagsHandler() did not return injected handler")
	}
}

func TestNewRepoTagsHandler_default(t *testing.T) {
	opts := &showTagsOptions{Format: option.Format{Type: option.FormatTypeText.Name}}
	got, err := newRepoTagsHandler(opts)
	if err != nil {
		t.Fatal(err)
	}
	if gotType, wantType := reflect.TypeOf(got), reflect.TypeOf(text.NewRepoTagsHandler(nil)); gotType != wantType {
		t.Errorf("handler type = %v, want %v", gotType, wantType)
	}
}

func TestRecordingRepoTagsHandler_callOrderAndError(t *testing.T) {
	wantedErr := errors.New("tag listing failed")
	handler := &recordingRepoTagsHandler{err: wantedErr}
	if err := handler.OnTagListed("v1"); !errors.Is(err, wantedErr) {
		t.Fatalf("OnTagListed() error = %v, want %v", err, wantedErr)
	}
	if got, want := handler.calls, []string{"OnTagListed"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("handler calls = %v, want %v", got, want)
	}
}
