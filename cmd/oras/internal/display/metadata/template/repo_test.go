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

func TestNewRepoListHandler(t *testing.T) {
	if got := NewRepoListHandler(&bytes.Buffer{}, "{{.registry}}", "localhost:5000"); got == nil {
		t.Fatal("NewRepoListHandler() returned nil")
	}
}

func TestRepoListHandler_Render(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewRepoListHandler(buf, "{{.registry}}|{{range .repositories}}{{.}} {{end}}", "localhost:5000")

	for _, repo := range []string{"team/alpha", "team/beta"} {
		if err := handler.OnRepositoryListed(repo); err != nil {
			t.Fatalf("OnRepositoryListed(%q) error = %v, want nil", repo, err)
		}
	}
	if err := handler.Render(); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	want := "localhost:5000|team/alpha team/beta "
	if got := buf.String(); got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

// With nothing listed the model still renders, with an empty repository list.
func TestRepoListHandler_Render_empty(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewRepoListHandler(buf, "{{len .repositories}}", "localhost:5000")

	if err := handler.Render(); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}
	if got, want := buf.String(), "0"; got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestRepoListHandler_Render_invalidTemplate(t *testing.T) {
	handler := NewRepoListHandler(&bytes.Buffer{}, "{{.registry", "localhost:5000")
	if err := handler.Render(); err == nil {
		t.Error("Render() error = nil, want an error for an unparsable template")
	}
}

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
