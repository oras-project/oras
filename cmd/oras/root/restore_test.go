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
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/oras-project/oras-go/v3"
	"github.com/oras-project/oras-go/v3/content/memory"
	"github.com/oras-project/oras-go/v3/content/oci"
	"github.com/oras-project/oras-go/v3/errdef"
	"oras.land/oras/cmd/oras/internal/display/displaytest"
	orasio "oras.land/oras/internal/io"
)

func newRestoreTestTar(t *testing.T) string {
	t.Helper()
	return newRestoreTestTarWithRefs(t, "v1")
}

// newRestoreTestTarWithRefs creates a backup tar archive holding one manifest
// for each ref, stored under that ref in org.opencontainers.image.ref.name.
func newRestoreTestTarWithRefs(t *testing.T, refs ...string) string {
	t.Helper()
	ctx := context.Background()
	layoutDir := t.TempDir()
	layout, err := oci.New(layoutDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range refs {
		desc, err := oras.PackManifest(ctx, layout, oras.PackManifestVersion1_1, "application/example", oras.PackManifestOptions{
			ManifestAnnotations: map[string]string{"ref": ref},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := layout.Tag(ctx, desc, ref); err != nil {
			t.Fatal(err)
		}
	}
	tarPath := filepath.Join(t.TempDir(), "backup.tar")
	tarFile, err := os.Create(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tarFile.Close() }()
	if err := orasio.TarDirectory(tarFile, layoutDir); err != nil {
		t.Fatal(err)
	}
	return tarPath
}

func Test_doRestore_handlerCalls(t *testing.T) {
	opts := &restoreOptions{input: newRestoreTestTar(t), repository: "example"}
	recorder := &displaytest.CallRecorder{}
	statusHandler := &displaytest.RecordingRestoreStatusHandler{Recorder: recorder}
	metadataHandler := &displaytest.RecordingRestoreMetadataHandler{Recorder: recorder}

	if err := doRestore(context.Background(), opts, memory.New(), time.Now(), statusHandler, metadataHandler); err != nil {
		t.Fatal(err)
	}

	calls := recorder.Snapshot()
	if len(calls) < 6 {
		t.Fatalf("handler calls = %v, want at least 6", calls)
	}
	if got, want := calls[:3], []string{"OnTarLoaded", "OnTagsFound", "StartTracking"}; !reflect.DeepEqual(got, want) {
		t.Errorf("first handler calls = %v, want %v", got, want)
	}
	if got, want := calls[len(calls)-3:], []string{"StopTracking", "OnArtifactPushed", "OnRestoreCompleted"}; !reflect.DeepEqual(got, want) {
		t.Errorf("last handler calls = %v, want %v", got, want)
	}
}

func Test_doRestore_dryRun(t *testing.T) {
	opts := &restoreOptions{input: newRestoreTestTar(t), repository: "example", dryRun: true}
	recorder := &displaytest.CallRecorder{}
	statusHandler := &displaytest.RecordingRestoreStatusHandler{Recorder: recorder}
	metadataHandler := &displaytest.RecordingRestoreMetadataHandler{Recorder: recorder}

	if err := doRestore(context.Background(), opts, memory.New(), time.Now(), statusHandler, metadataHandler); err != nil {
		t.Fatal(err)
	}

	want := []string{"OnTarLoaded", "OnTagsFound", "OnArtifactPushed", "OnRestoreCompleted"}
	if got := recorder.Snapshot(); !reflect.DeepEqual(got, want) {
		t.Errorf("handler calls = %v, want %v", got, want)
	}
}

func Test_doRestore_artifactPushedError(t *testing.T) {
	opts := &restoreOptions{input: newRestoreTestTar(t), repository: "example"}
	wantErr := errors.New("artifact pushed failed")
	recorder := &displaytest.CallRecorder{}
	statusHandler := &displaytest.RecordingRestoreStatusHandler{Recorder: recorder}
	metadataHandler := &displaytest.RecordingRestoreMetadataHandler{Recorder: recorder, ArtifactPushedErr: wantErr}

	err := doRestore(context.Background(), opts, memory.New(), time.Now(), statusHandler, metadataHandler)
	if !errors.Is(err, wantErr) {
		t.Fatalf("doRestore() error = %v, want %v", err, wantErr)
	}
	if calls := recorder.Snapshot(); slices.Contains(calls, "OnRestoreCompleted") {
		t.Errorf("handler calls = %v, want no OnRestoreCompleted after an error", calls)
	}
}

func Test_doRestore_fullReferenceBackup(t *testing.T) {
	tests := []struct {
		name     string
		refs     []string
		tags     []string
		wantTags []string
	}{
		{
			name:     "full references, all tags",
			refs:     []string{"localhost:5000/example:v1", "localhost:5000/example:v2"},
			wantTags: []string{"v1", "v2"},
		},
		{
			name:     "full references, specified tag",
			refs:     []string{"localhost:5000/example:v1", "localhost:5000/example:v2"},
			tags:     []string{"v2"},
			wantTags: []string{"v2"},
		},
		{
			name:     "tag-only backup from an earlier version",
			refs:     []string{"v1"},
			tags:     []string{"v1"},
			wantTags: []string{"v1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			opts := &restoreOptions{input: newRestoreTestTarWithRefs(t, tt.refs...), repository: "registry.example.com/example", tags: tt.tags}
			recorder := &displaytest.CallRecorder{}
			statusHandler := &displaytest.RecordingRestoreStatusHandler{Recorder: recorder}
			metadataHandler := &displaytest.RecordingRestoreMetadataHandler{Recorder: recorder}
			dst := memory.New()

			if err := doRestore(ctx, opts, dst, time.Now(), statusHandler, metadataHandler); err != nil {
				t.Fatal(err)
			}

			for _, tag := range tt.wantTags {
				if _, err := dst.Resolve(ctx, tag); err != nil {
					t.Errorf("tag %q not restored: %v", tag, err)
				}
			}
			for _, ref := range tt.refs {
				if slices.Contains(tt.wantTags, ref) {
					continue
				}
				if _, err := dst.Resolve(ctx, ref); err == nil {
					t.Errorf("stored reference %q was pushed as a tag", ref)
				}
			}
		})
	}
}

func Test_resolveRestoreRefs(t *testing.T) {
	ctx := context.Background()
	newLayout := func(t *testing.T, refs ...string) oras.ReadOnlyTarget {
		t.Helper()
		layout, err := oci.NewFromTar(ctx, newRestoreTestTarWithRefs(t, refs...))
		if err != nil {
			t.Fatal(err)
		}
		return layout
	}

	t.Run("maps stored references to tags", func(t *testing.T) {
		srcRefs, tags, roots, err := resolveRestoreRefs(ctx, newLayout(t, "localhost:5000/example:v1"), nil)
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{"localhost:5000/example:v1"}; !reflect.DeepEqual(srcRefs, want) {
			t.Errorf("srcRefs = %v, want %v", srcRefs, want)
		}
		if want := []string{"v1"}; !reflect.DeepEqual(tags, want) {
			t.Errorf("tags = %v, want %v", tags, want)
		}
		if len(roots) != 1 {
			t.Errorf("roots = %v, want 1 descriptor", roots)
		}
	})

	t.Run("specified tag not in backup", func(t *testing.T) {
		_, _, _, err := resolveRestoreRefs(ctx, newLayout(t, "localhost:5000/example:v1"), []string{"v2"})
		if !errors.Is(err, errdef.ErrNotFound) {
			t.Errorf("error = %v, want %v", err, errdef.ErrNotFound)
		}
	})

	t.Run("two references restore to the same tag", func(t *testing.T) {
		_, _, _, err := resolveRestoreRefs(ctx, newLayout(t, "localhost:5000/a:v1", "localhost:5000/b:v1"), nil)
		if err == nil {
			t.Error("expected an error for references sharing a tag")
		}
	})
}

func Test_tagFromRefName(t *testing.T) {
	tests := []struct {
		name    string
		refName string
		want    string
		wantErr bool
	}{
		{name: "tag only", refName: "v1.0", want: "v1.0"},
		{name: "full reference", refName: "docker.io/library/alpine:3.19", want: "3.19"},
		{name: "registry with port", refName: "localhost:5000/repo:v1", want: "v1"},
		{name: "nested repository", refName: "registry.example.com/a/b/c:latest", want: "latest"},
		{name: "digest reference", refName: "localhost:5000/repo@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412", wantErr: true},
		{name: "tag and digest", refName: "localhost:5000/repo:v1@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412", wantErr: true},
		{name: "repository without tag", refName: "localhost:5000/repo", wantErr: true},
		{name: "invalid reference", refName: "localhost:5000/Repo:v1", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tagFromRefName(tt.refName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("tagFromRefName(%q) error = %v, wantErr %v", tt.refName, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("tagFromRefName(%q) = %q, want %q", tt.refName, got, tt.want)
			}
		})
	}
}
