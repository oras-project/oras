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
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
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

func TestNewRepoTagsHandler_invalidFormat(t *testing.T) {
	_, err := newRepoTagsHandler(&showTagsOptions{Format: option.Format{Type: "invalid"}})
	if err == nil {
		t.Fatal("newRepoTagsHandler() error = nil, want invalid format error")
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

func TestShowTags_injectedHandler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/example/tags/list" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"name":"example","tags":["v1"]}`))
	}))
	defer server.Close()
	host := strings.TrimPrefix(server.URL, "http://")
	opts := &showTagsOptions{Target: option.Target{Type: option.TargetTypeRemote, RawReference: host + "/example"}}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	opts.Remote.ApplyFlags(cmd.Flags())
	if err := cmd.Flags().Set("plain-http", "true"); err != nil {
		t.Fatal(err)
	}
	handler := &recordingRepoTagsHandler{}
	if err := showTags(cmd, opts, handler); err != nil {
		t.Fatal(err)
	}
	if got, want := handler.calls, []string{"OnTagListed", "Render"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("handler calls = %v, want %v", got, want)
	}
}
