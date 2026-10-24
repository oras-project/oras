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
	"errors"
	"reflect"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata/text"
	"oras.land/oras/cmd/oras/internal/option"
)

type recordingResolveHandler struct {
	calls []string
	err   error
}

func (h *recordingResolveHandler) OnResolved(ocispec.Descriptor) error {
	h.calls = append(h.calls, "OnResolved")
	return h.err
}

func TestNewResolveHandler_injected(t *testing.T) {
	want := &recordingResolveHandler{}
	got := newResolveHandler(&resolveOptions{}, want)
	if got != want {
		t.Fatal("newResolveHandler() did not return injected handler")
	}
}

func TestNewResolveHandler_default(t *testing.T) {
	opts := &resolveOptions{Common: option.Common{}, fullRef: true}
	got := newResolveHandler(opts)
	if reflect.TypeOf(got) != reflect.TypeOf(text.NewResolveHandler(nil, true, "")) {
		t.Errorf("handler type = %T, want text resolve handler", got)
	}
}

func TestRecordingResolveHandler_error(t *testing.T) {
	wantedErr := errors.New("resolve output failed")
	handler := &recordingResolveHandler{err: wantedErr}
	if err := handler.OnResolved(ocispec.Descriptor{}); !errors.Is(err, wantedErr) {
		t.Fatalf("OnResolved() error = %v, want %v", err, wantedErr)
	}
	if got, want := handler.calls, []string{"OnResolved"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("handler calls = %v, want %v", got, want)
	}
}
