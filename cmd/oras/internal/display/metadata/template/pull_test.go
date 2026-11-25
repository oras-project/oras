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
	"path/filepath"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/option"
)

func TestNewPullHandler(t *testing.T) {
	if got := NewPullHandler(&bytes.Buffer{}, "localhost:5000/test", "{{.reference}}"); got == nil {
		t.Fatal("NewPullHandler() returned nil")
	}
}

func TestPullHandler_OnPulled(t *testing.T) {
	handler := NewPullHandler(&bytes.Buffer{}, "localhost:5000/test", "{{.reference}}").(*PullHandler)

	handler.OnPulled(&option.Target{Path: "ignored.example/test"}, testDescriptor)
	if got := handler.root.Digest; got != testDescriptor.Digest {
		t.Errorf("OnPulled() digest = %q, want %q", got, testDescriptor.Digest)
	}
	if got := handler.root.MediaType; got != testDescriptor.MediaType {
		t.Errorf("OnPulled() media type = %q, want %q", got, testDescriptor.MediaType)
	}
}

func TestPullHandler_Render(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewPullHandler(
		buf,
		"localhost:5000/test",
		"{{.reference}}|{{range .files}}{{.path}}={{.reference}};{{end}}",
	).(*PullHandler)

	handler.OnPulled(&option.Target{Path: "ignored.example/test"}, testDescriptor)
	outputDir := t.TempDir()
	fileDescriptor := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
		Digest:    "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		Size:      42,
		Annotations: map[string]string{
			ocispec.AnnotationTitle: "file.txt",
		},
	}
	if err := handler.OnFilePulled("file.txt", outputDir, fileDescriptor, "layer/file.txt"); err != nil {
		t.Fatalf("OnFilePulled() error = %v, want nil", err)
	}

	if err := handler.Render(); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}
	want := "localhost:5000/test@" + testDigest + "|" +
		filepath.Join(outputDir, "file.txt") + "=layer/file.txt@" +
		fileDescriptor.Digest.String() + ";"
	if got := buf.String(); got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestPullHandler_Render_noFiles(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewPullHandler(buf, "localhost:5000/test", "{{.reference}}|{{range .files}}{{.path}};{{end}}").(*PullHandler)
	handler.OnPulled(nil, testDescriptor)

	if err := handler.Render(); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}
	want := "localhost:5000/test@" + testDigest + "|"
	if got := buf.String(); got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestPullHandler_OnFilePulled(t *testing.T) {
	outputDir := t.TempDir()
	handler := NewPullHandler(&bytes.Buffer{}, "unused", "{{.reference}}").(*PullHandler)
	fileDescriptor := testDescriptor
	fileDescriptor.Annotations = map[string]string{
		ocispec.AnnotationTitle: "title.txt",
	}

	if err := handler.OnFilePulled("title.txt", outputDir, fileDescriptor, "layer/title.txt"); err != nil {
		t.Fatalf("OnFilePulled() with title error = %v, want nil", err)
	}
	absoluteName := filepath.Join(outputDir, "untitled.txt")
	if err := handler.OnFilePulled(absoluteName, "unused", testDescriptor, "layer/untitled.txt"); err != nil {
		t.Fatalf("OnFilePulled() without title error = %v, want nil", err)
	}

	files := handler.pulled.Files()
	if len(files) != 2 {
		t.Fatalf("OnFilePulled() recorded %d files, want 2", len(files))
	}
	if got, want := files[0].Path, filepath.Join(outputDir, "title.txt"); got != want {
		t.Errorf("OnFilePulled() title path = %q, want %q", got, want)
	}
	if got, want := files[0].Reference, "layer/title.txt@"+testDigest; got != want {
		t.Errorf("OnFilePulled() title reference = %q, want %q", got, want)
	}
	if got := files[0].Annotations[ocispec.AnnotationTitle]; got != "title.txt" {
		t.Errorf("OnFilePulled() title annotation = %q, want %q", got, "title.txt")
	}
	if got, want := files[1].Path, absoluteName; got != want {
		t.Errorf("OnFilePulled() no-title path = %q, want %q", got, want)
	}
	if got, want := files[1].Reference, "layer/untitled.txt@"+testDigest; got != want {
		t.Errorf("OnFilePulled() no-title reference = %q, want %q", got, want)
	}
	if files[1].Annotations != nil {
		t.Errorf("OnFilePulled() no-title annotations = %v, want nil", files[1].Annotations)
	}
}

func TestPullHandler_Render_invalidTemplate(t *testing.T) {
	handler := NewPullHandler(&bytes.Buffer{}, "localhost:5000/test", "{{.reference").(*PullHandler)
	handler.OnPulled(nil, testDescriptor)

	if err := handler.Render(); err == nil {
		t.Error("Render() error = nil, want an error for an unparsable template")
	}
}

func TestPullHandler_OnLayerSkipped(t *testing.T) {
	handler := NewPullHandler(&bytes.Buffer{}, "localhost:5000/test", "{{.reference}}")
	if err := handler.OnLayerSkipped(testDescriptor); err != nil {
		t.Errorf("OnLayerSkipped() error = %v, want nil", err)
	}
}

// {{len .files}} must work after a pull that produced no files, as it does for
// the tags of repo tags and the repositories of repo ls.
func TestPullHandler_Render_noFilesLen(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewPullHandler(buf, "localhost:5000/test", "{{len .files}}").(*PullHandler)
	handler.OnPulled(nil, testDescriptor)

	if err := handler.Render(); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}
	if got, want := buf.String(), "0"; got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}
