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
	"strings"
	"testing"
	"time"

	"github.com/oras-project/oras-go/v3"
	"github.com/oras-project/oras-go/v3/content/memory"
	"github.com/oras-project/oras-go/v3/content/oci"
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

// newRestoreTestTargets returns a function that creates an in-memory
// repository for each target repository, and the repositories it created.
func newRestoreTestTargets() (func(string) (oras.GraphTarget, error), map[string]*memory.Store) {
	stores := make(map[string]*memory.Store)
	return func(repo string) (oras.GraphTarget, error) {
		store, ok := stores[repo]
		if !ok {
			store = memory.New()
			stores[repo] = store
		}
		return store, nil
	}, stores
}

func Test_doRestore_handlerCalls(t *testing.T) {
	opts := &restoreOptions{input: newRestoreTestTar(t), target: "localhost:5000/example"}
	recorder := &displaytest.CallRecorder{}
	statusHandler := &displaytest.RecordingRestoreStatusHandler{Recorder: recorder}
	metadataHandler := &displaytest.RecordingRestoreMetadataHandler{Recorder: recorder}

	newTarget, _ := newRestoreTestTargets()
	if err := doRestore(context.Background(), opts, newTarget, time.Now(), statusHandler, metadataHandler); err != nil {
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
	opts := &restoreOptions{input: newRestoreTestTar(t), target: "localhost:5000/example", dryRun: true}
	recorder := &displaytest.CallRecorder{}
	statusHandler := &displaytest.RecordingRestoreStatusHandler{Recorder: recorder}
	metadataHandler := &displaytest.RecordingRestoreMetadataHandler{Recorder: recorder}

	newTarget, _ := newRestoreTestTargets()
	if err := doRestore(context.Background(), opts, newTarget, time.Now(), statusHandler, metadataHandler); err != nil {
		t.Fatal(err)
	}

	want := []string{"OnTarLoaded", "OnTagsFound", "OnArtifactPushed", "OnRestoreCompleted"}
	if got := recorder.Snapshot(); !reflect.DeepEqual(got, want) {
		t.Errorf("handler calls = %v, want %v", got, want)
	}
}

func Test_doRestore_artifactPushedError(t *testing.T) {
	opts := &restoreOptions{input: newRestoreTestTar(t), target: "localhost:5000/example"}
	wantErr := errors.New("artifact pushed failed")
	recorder := &displaytest.CallRecorder{}
	statusHandler := &displaytest.RecordingRestoreStatusHandler{Recorder: recorder}
	metadataHandler := &displaytest.RecordingRestoreMetadataHandler{Recorder: recorder, ArtifactPushedErr: wantErr}

	newTarget, _ := newRestoreTestTargets()
	err := doRestore(context.Background(), opts, newTarget, time.Now(), statusHandler, metadataHandler)
	if !errors.Is(err, wantErr) {
		t.Fatalf("doRestore() error = %v, want %v", err, wantErr)
	}
	if calls := recorder.Snapshot(); slices.Contains(calls, "OnRestoreCompleted") {
		t.Errorf("handler calls = %v, want no OnRestoreCompleted after an error", calls)
	}
}

func Test_doRestore_fullReferenceBackup(t *testing.T) {
	tests := []struct {
		name   string
		refs   []string
		target string
		// target repository -> restored tags
		want map[string][]string
	}{
		{
			name:   "full references, all tags",
			refs:   []string{"localhost:5000/example:v1", "localhost:5000/example:v2"},
			target: "registry.example.com/ns",
			want:   map[string][]string{"registry.example.com/ns/example": {"v1", "v2"}},
		},
		{
			name:   "multiple repositories to a registry",
			refs:   []string{"docker.io/library/redis:7.2", "ghcr.io/oras-project/oras:v1"},
			target: "registry.example.com",
			want: map[string][]string{
				"registry.example.com/library/redis":     {"7.2"},
				"registry.example.com/oras-project/oras": {"v1"},
			},
		},
		{
			name:   "same tag in multiple repositories",
			refs:   []string{"localhost:5000/a:v1", "localhost:5000/b:v1"},
			target: "registry.example.com/ns",
			want: map[string][]string{
				"registry.example.com/ns/a": {"v1"},
				"registry.example.com/ns/b": {"v1"},
			},
		},
		{
			name:   "tag-only backup from an earlier version",
			refs:   []string{"v1"},
			target: "registry.example.com/example",
			want:   map[string][]string{"registry.example.com/example": {"v1"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			opts := &restoreOptions{input: newRestoreTestTarWithRefs(t, tt.refs...), target: tt.target}
			recorder := &displaytest.CallRecorder{}
			statusHandler := &displaytest.RecordingRestoreStatusHandler{Recorder: recorder}
			metadataHandler := &displaytest.RecordingRestoreMetadataHandler{Recorder: recorder}
			newTarget, stores := newRestoreTestTargets()

			if err := doRestore(ctx, opts, newTarget, time.Now(), statusHandler, metadataHandler); err != nil {
				t.Fatal(err)
			}

			if len(stores) != len(tt.want) {
				t.Errorf("restored to %d repositories, want %d", len(stores), len(tt.want))
			}
			for repo, tags := range tt.want {
				store, ok := stores[repo]
				if !ok {
					t.Errorf("nothing restored to %q", repo)
					continue
				}
				for _, tag := range tags {
					if _, err := store.Resolve(ctx, tag); err != nil {
						t.Errorf("tag %q not restored to %q: %v", tag, repo, err)
					}
				}
				for _, ref := range tt.refs {
					if slices.Contains(tags, ref) {
						continue
					}
					if _, err := store.Resolve(ctx, ref); err == nil {
						t.Errorf("stored reference %q was pushed as a tag to %q", ref, repo)
					}
				}
			}
		})
	}
}

func Test_doRestore_collision(t *testing.T) {
	refs := []string{"docker.io/library/nginx:1.25", "mirror.gcr.io/library/nginx:1.25"}
	for _, dryRun := range []bool{false, true} {
		opts := &restoreOptions{input: newRestoreTestTarWithRefs(t, refs...), target: "registry.example.com", dryRun: dryRun}
		recorder := &displaytest.CallRecorder{}
		statusHandler := &displaytest.RecordingRestoreStatusHandler{Recorder: recorder}
		metadataHandler := &displaytest.RecordingRestoreMetadataHandler{Recorder: recorder}
		newTarget, stores := newRestoreTestTargets()

		if err := doRestore(context.Background(), opts, newTarget, time.Now(), statusHandler, metadataHandler); err == nil {
			t.Errorf("dryRun=%v: expected an error for two repositories restoring to the same target", dryRun)
		}
		if len(stores) != 0 {
			t.Errorf("dryRun=%v: restored to %v, want nothing pushed", dryRun, stores)
		}
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
		srcRefs, tags, roots, err := resolveRestoreRefs(ctx, newLayout(t, "localhost:5000/example:v1"))
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

	t.Run("same tag in two repositories", func(t *testing.T) {
		srcRefs, tags, roots, err := resolveRestoreRefs(ctx, newLayout(t, "localhost:5000/a:v1", "localhost:5000/b:v1"))
		if err != nil {
			t.Fatal(err)
		}
		slices.Sort(srcRefs)
		if want := []string{"localhost:5000/a:v1", "localhost:5000/b:v1"}; !reflect.DeepEqual(srcRefs, want) {
			t.Errorf("srcRefs = %v, want %v", srcRefs, want)
		}
		if want := []string{"v1", "v1"}; !reflect.DeepEqual(tags, want) {
			t.Errorf("tags = %v, want %v", tags, want)
		}
		if len(roots) != 2 {
			t.Errorf("roots = %v, want 2 descriptors", roots)
		}
	})

	t.Run("invalid entry fails restoring all tags", func(t *testing.T) {
		_, _, _, err := resolveRestoreRefs(ctx, newLayout(t, "localhost:5000/example:v1", "localhost:5000/example"))
		if err == nil {
			t.Error("expected an error for an entry without a tag")
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
		{name: "invalid tag only", refName: "invalid+tag", wantErr: true},
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

func Test_restoreTargets(t *testing.T) {
	tests := []struct {
		name                   string
		target                 string
		srcRefs                []string
		preserveSourceRegistry bool
		want                   []string
		wantErr                bool
	}{
		{
			name:    "registry target",
			target:  "harbor.local",
			srcRefs: []string{"docker.io/library/redis:7.2", "ghcr.io/oras-project/oras:v1.3.0"},
			want:    []string{"harbor.local/library/redis", "harbor.local/oras-project/oras"},
		},
		{
			name:    "namespace target",
			target:  "harbor.local/myproject",
			srcRefs: []string{"docker.io/library/redis:7.2"},
			want:    []string{"harbor.local/myproject/library/redis"},
		},
		{
			name:    "single repository is not restored to the target itself",
			target:  "harbor.local/myproject/redis",
			srcRefs: []string{"docker.io/library/redis:7.2"},
			want:    []string{"harbor.local/myproject/redis/library/redis"},
		},
		{
			name:    "registry with port",
			target:  "localhost:5000",
			srcRefs: []string{"localhost:5001/repo:v1", "localhost:5001/repo:v2"},
			want:    []string{"localhost:5000/repo", "localhost:5000/repo"},
		},
		{
			name:                   "preserve source registry",
			target:                 "harbor.local",
			srcRefs:                []string{"docker.io/library/nginx:1.25", "mirror.gcr.io/library/nginx:1.25"},
			preserveSourceRegistry: true,
			want:                   []string{"harbor.local/docker.io/library/nginx", "harbor.local/mirror.gcr.io/library/nginx"},
		},
		{
			name:                   "preserve source registry with a port",
			target:                 "harbor.local",
			srcRefs:                []string{"localhost:5000/repo:v1"},
			preserveSourceRegistry: true,
			want:                   []string{"harbor.local/localhost_5000/repo"},
		},
		{
			name:                   "preserve source registry with an uppercase host",
			target:                 "harbor.local",
			srcRefs:                []string{"Registry.Example.com/app:v1"},
			preserveSourceRegistry: true,
			want:                   []string{"harbor.local/registry.example.com/app"},
		},
		{
			name:                   "preserve source registry with an IPv6 literal host",
			target:                 "harbor.local",
			srcRefs:                []string{"[::1]:5000/app:v1"},
			preserveSourceRegistry: true,
			wantErr:                true,
		},
		{
			name:    "hosts with and without a port collide",
			target:  "harbor.local",
			srcRefs: []string{"localhost:5000/app:v1", "registry.example.com/app:v1"},
			wantErr: true,
		},
		{
			name:                   "hosts with and without a port with preserve source registry",
			target:                 "harbor.local",
			srcRefs:                []string{"localhost:5000/app:v1", "registry.example.com/app:v1"},
			preserveSourceRegistry: true,
			want:                   []string{"harbor.local/localhost_5000/app", "harbor.local/registry.example.com/app"},
		},
		{
			name:                   "encoded host collides with a host using '_'",
			target:                 "harbor.local",
			srcRefs:                []string{"a:5000/app:v1", "a_5000/app:v1"},
			preserveSourceRegistry: true,
			wantErr:                true,
		},
		{
			name:    "target repository of 255 characters",
			target:  "harbor.local/ns",
			srcRefs: []string{"registry.example.com/" + strings.Repeat("a", 252) + ":v1"},
			want:    []string{"harbor.local/ns/" + strings.Repeat("a", 252)},
		},
		{
			name:                   "target repository longer than 255 characters",
			target:                 "harbor.local/ns",
			srcRefs:                []string{"registry.example.com/" + strings.Repeat("a", 240) + ":v1"},
			preserveSourceRegistry: true,
			wantErr:                true,
		},
		{
			name:    "collision",
			target:  "harbor.local",
			srcRefs: []string{"docker.io/library/nginx:1.25", "mirror.gcr.io/library/nginx:1.25"},
			wantErr: true,
		},
		{
			name:    "multiple collisions",
			target:  "harbor.local",
			srcRefs: []string{"docker.io/library/nginx:1.25", "mirror.gcr.io/library/nginx:1.25", "mirror.gcr.io/library/nginx:1.26", "docker.io/library/redis:7.2", "mirror.gcr.io/library/redis:7.2"},
			wantErr: true,
		},
		{
			name:    "tag-only backup from an earlier version",
			target:  "harbor.local/myproject/redis",
			srcRefs: []string{"v1", "v2"},
			want:    []string{"harbor.local/myproject/redis", "harbor.local/myproject/redis"},
		},
		{
			name:    "tag-only backup to a registry",
			target:  "harbor.local",
			srcRefs: []string{"v1"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := restoreTargets(tt.target, tt.srcRefs, tt.preserveSourceRegistry)
			if (err != nil) != tt.wantErr {
				t.Fatalf("restoreTargets() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("restoreTargets() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_restoreTargets_reportsEveryCollision(t *testing.T) {
	srcRefs := []string{
		"docker.io/library/nginx:1.25",
		"mirror.gcr.io/library/nginx:1.25",
		"mirror.gcr.io/library/nginx:1.26", // a second tag of a colliding repository is not listed again
		"docker.io/library/redis:7.2",
		"mirror.gcr.io/library/redis:7.2",
	}
	_, err := restoreTargets("harbor.local", srcRefs, false)
	if err == nil {
		t.Fatal("expected an error for colliding repositories")
	}
	for _, want := range []string{
		`"docker.io/library/nginx" and "mirror.gcr.io/library/nginx" both restore to "harbor.local/library/nginx"`,
		`"docker.io/library/redis" and "mirror.gcr.io/library/redis" both restore to "harbor.local/library/redis"`,
	} {
		if got := strings.Count(err.Error(), want); got != 1 {
			t.Errorf("error %q lists %q %d time(s), want 1", err, want, got)
		}
	}
}

func Test_restoreTargets_preserveSourceRegistryWithPort(t *testing.T) {
	got, err := restoreTargets("harbor.local", []string{"localhost:5000/repo:v1"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"harbor.local/localhost_5000/repo"}; !reflect.DeepEqual(got, want) {
		t.Errorf("restoreTargets() = %v, want %v", got, want)
	}

	_, err = restoreTargets("harbor.local", []string{"[::1]:5000/repo:v1"}, true)
	if err == nil {
		t.Fatal("expected an error for an IPv6 literal source registry")
	}
	if !strings.Contains(err.Error(), `IPv6 literal host "[::1]:5000"`) {
		t.Errorf("error = %q, want it to name the IPv6 literal host", err)
	}
}

func Test_encodeRegistryComponent(t *testing.T) {
	tests := []struct {
		host    string
		want    string
		wantErr bool
	}{
		{host: "docker.io", want: "docker.io"},
		{host: "Registry.Example.com", want: "registry.example.com"},
		{host: "registry.lan:5000", want: "registry.lan_5000"},
		{host: "localhost:5000", want: "localhost_5000"},
		{host: "[::1]:5000", wantErr: true},
		{host: "[::1]", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got, err := encodeRegistryComponent(tt.host)
			if (err != nil) != tt.wantErr {
				t.Fatalf("encodeRegistryComponent(%q) error = %v, wantErr %v", tt.host, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("encodeRegistryComponent(%q) = %q, want %q", tt.host, got, tt.want)
			}
		})
	}
}

func Test_parseRestoreTarget(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		want    string
		wantErr bool
	}{
		{name: "registry", target: "harbor.local", want: "harbor.local"},
		{name: "registry with port", target: "localhost:5000", want: "localhost:5000"},
		{name: "namespace", target: "harbor.local/myproject", want: "harbor.local/myproject"},
		{name: "registry with port and namespace", target: "localhost:5000/a/b", want: "localhost:5000/a/b"},
		{name: "namespace with tags", target: "harbor.local/myproject:v1,v2", wantErr: true},
		{name: "namespace with tag", target: "harbor.local/myproject:v1", wantErr: true},
		{name: "empty", target: "", wantErr: true},
		{name: "invalid registry", target: "harbor local", wantErr: true},
		{name: "invalid namespace", target: "harbor.local/MyProject", wantErr: true},
		{name: "digest", target: "harbor.local/myproject@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRestoreTarget(tt.target)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseRestoreTarget(%q) error = %v, wantErr %v", tt.target, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("parseRestoreTarget(%q) = %q, want %q", tt.target, got, tt.want)
			}
		})
	}
}
