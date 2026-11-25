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

package template

import (
	"bytes"
	"testing"
)

func TestNewRepoTagsHandler(t *testing.T) {
	if got := NewRepoTagsHandler(&bytes.Buffer{}, "{{.tags}}"); got == nil {
		t.Fatal("NewRepoTagsHandler() returned nil")
	}
}

func TestRepoTagsHandler_Render(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewRepoTagsHandler(buf, "{{range .tags}}{{.}} {{end}}")

	for _, tag := range []string{"v1.0.0", "latest"} {
		if err := handler.OnTagListed(tag); err != nil {
			t.Fatalf("OnTagListed(%q) error = %v, want nil", tag, err)
		}
	}
	if err := handler.Render(); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	want := "v1.0.0 latest "
	if got := buf.String(); got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestRepoTagsHandler_Render_empty(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewRepoTagsHandler(buf, "{{len .tags}}")

	if err := handler.Render(); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}
	if got, want := buf.String(), "0"; got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestRepoTagsHandler_Render_invalidTemplate(t *testing.T) {
	handler := NewRepoTagsHandler(&bytes.Buffer{}, "{{.tags")
	if err := handler.Render(); err == nil {
		t.Error("Render() error = nil, want an error for an unparsable template")
	}
}
