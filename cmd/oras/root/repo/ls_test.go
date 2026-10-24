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

type recordingRepoListHandler struct {
	calls []string
	err   error
}

func (h *recordingRepoListHandler) OnRepositoryListed(repo string) error {
	h.calls = append(h.calls, repo)
	return h.err
}

func (h *recordingRepoListHandler) Render() error {
	h.calls = append(h.calls, "Render")
	return nil
}

func TestNewRepoListHandler_injected(t *testing.T) {
	want := &recordingRepoListHandler{}
	got, err := newRepoListHandler(&repositoryOptions{}, "registry.example", want)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatal("newRepoListHandler() did not return injected handler")
	}
}

func TestNewRepoListHandler_default(t *testing.T) {
	opts := &repositoryOptions{Format: option.Format{Type: option.FormatTypeText.Name}}
	got, err := newRepoListHandler(opts, "registry.example")
	if err != nil {
		t.Fatal(err)
	}
	if gotType, wantType := reflect.TypeOf(got), reflect.TypeOf(text.NewRepoListHandler(nil, "")); gotType != wantType {
		t.Errorf("handler type = %v, want %v", gotType, wantType)
	}
}

func TestEmitRepositories_filtersAndStopsOnError(t *testing.T) {
	wantedErr := errors.New("listing failed")
	handler := &recordingRepoListHandler{err: wantedErr}
	err := emitRepositories([]string{"example/a", "other/b"}, "example/", handler)
	if !errors.Is(err, wantedErr) {
		t.Fatalf("emitRepositories() error = %v, want %v", err, wantedErr)
	}
	if got, want := handler.calls, []string{"example/a"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("handler calls = %v, want %v", got, want)
	}
}
