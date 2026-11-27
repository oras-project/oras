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
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oras-project/oras-go/v3/content/oci"
	"github.com/oras-project/oras-go/v3/errdef"
)

func TestOCILayoutPathExtraTags(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	layout := filepath.Join(dir, "layout")
	mirror := filepath.Join(dir, "mirror")
	input := filepath.Join(dir, "app.txt")
	manifest := filepath.Join(dir, "manifest.json")
	app := "localhost:5000/team/app"
	other := "example.com/team/other"

	run := func(args ...string) string {
		t.Helper()
		var out bytes.Buffer
		cmd := New()
		cmd.SetOut(&out)
		cmd.SetArgs(args)
		if err := cmd.ExecuteContext(ctx); err != nil {
			t.Fatalf("oras %v: %v", args, err)
		}
		return out.String()
	}
	writeInput := func(data string) {
		t.Helper()
		if err := os.WriteFile(input, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	checkAliases := func(path, primary string, aliases ...string) string {
		t.Helper()
		store, err := oci.New(path)
		if err != nil {
			t.Fatal(err)
		}
		want, err := store.Resolve(ctx, primary)
		if err != nil {
			t.Fatal(err)
		}
		for _, alias := range aliases {
			got, err := store.Resolve(ctx, alias)
			if err != nil {
				t.Errorf("resolve %q: %v", alias, err)
			} else if got.Digest != want.Digest {
				t.Errorf("%q resolves to %s, want %s", alias, got.Digest, want.Digest)
			}
		}
		return want.Digest.String()
	}

	writeInput("first artifact")
	run("push", "--oci-layout-path", layout, "--disable-path-validation", app+":v1,latest,stable", input)
	writeInput("second artifact")
	run("push", "--oci-layout-path", layout, "--disable-path-validation", other+":v1,latest", input)
	appDigest := checkAliases(layout, app+":v1", app+":latest", app+":stable")
	otherDigest := checkAliases(layout, other+":v1", other+":latest")
	if appDigest == otherDigest {
		t.Fatal("test artifacts unexpectedly have the same digest")
	}
	if got := run("repo", "tags", "--oci-layout-path", layout, app); got != "latest\nstable\nv1\n" {
		t.Errorf("repository tags = %q, want tags latest, stable, v1", got)
	}
	store, err := oci.New(layout)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Resolve(ctx, "latest"); !errors.Is(err, errdef.ErrNotFound) {
		t.Errorf("bare latest lookup = %v, want %v", err, errdef.ErrNotFound)
	}

	run("copy", "--from-oci-layout-path", layout, "--to-oci-layout-path", mirror, app+":v1", other+":v1,latest")
	if got := checkAliases(mirror, other+":v1", other+":latest"); got != appDigest {
		t.Errorf("copied artifact digest = %s, want %s", got, appDigest)
	}
	run("copy", "--from-oci-layout-path", layout, "--to-oci-layout-path", mirror, app+":v1", app+"@"+appDigest+",copied")
	run("copy", "--from-oci-layout-path", layout, "--to-oci-layout-path", mirror, app+":v1", app+":@"+appDigest+",copied-empty-tag")
	checkAliases(mirror, appDigest, app+":copied", app+":copied-empty-tag")

	run("manifest", "fetch", "--oci-layout-path", layout, app+":latest", "--output", manifest)
	run("manifest", "push", "--oci-layout-path", layout, app+":manifest,manifest-latest", manifest)
	checkAliases(layout, app+":manifest", app+":manifest-latest")
	run("manifest", "push", "--oci-layout-path", layout, "--descriptor", app+":descriptor,descriptor-latest", manifest)
	checkAliases(layout, app+":descriptor", app+":descriptor-latest")

	run("manifest", "index", "create", "--oci-layout-path", layout, app+":index,index-latest", app+":v1")
	checkAliases(layout, app+":index", app+":index-latest")
	run("manifest", "index", "update", "--oci-layout-path", layout, app+":index", "--artifact-type", "application/vnd.example.index", "--tag", "index-updated")
	checkAliases(layout, app+":index", app+":index-updated")

	run("push", "--oci-layout", layout+":legacy,legacy-alias", "--disable-path-validation", input)
	checkAliases(layout, "legacy", "legacy-alias")
	run("push", "--oci-layout-path", layout, "bare,bare-alias", "--disable-path-validation", input)
	checkAliases(layout, "bare", "bare-alias")
}

func TestOCILayoutPathTag(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	layout := filepath.Join(dir, "layout")
	input := filepath.Join(dir, "app.txt")
	app := "localhost:5000/team/app"
	other := "example.com/team/other"

	run := func(args ...string) string {
		t.Helper()
		var out bytes.Buffer
		cmd := New()
		cmd.SetOut(&out)
		cmd.SetArgs(args)
		if err := cmd.ExecuteContext(ctx); err != nil {
			t.Fatalf("oras %v: %v", args, err)
		}
		return out.String()
	}
	push := func(data string, args ...string) {
		t.Helper()
		if err := os.WriteFile(input, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		run(append(append([]string{"push", "--disable-path-validation"}, args...), input)...)
	}
	resolve := func(ref string) (string, error) {
		t.Helper()
		store, err := oci.New(layout)
		if err != nil {
			t.Fatal(err)
		}
		desc, err := store.Resolve(ctx, ref)
		return desc.Digest.String(), err
	}
	mustResolve := func(ref string) string {
		t.Helper()
		got, err := resolve(ref)
		if err != nil {
			t.Fatalf("resolve %q: %v", ref, err)
		}
		return got
	}

	push("first artifact", "--oci-layout-path", layout, app+":v1")
	push("second artifact", "--oci-layout-path", layout, other+":v1")
	appDigest, otherDigest := mustResolve(app+":v1"), mustResolve(other+":v1")
	if appDigest == otherDigest {
		t.Fatal("test artifacts unexpectedly have the same digest")
	}

	run("tag", "--oci-layout-path", layout, app+":v1", "latest", "stable")
	run("tag", "--oci-layout-path", layout, other+":v1", "latest")

	for ref, want := range map[string]string{
		app + ":latest":   appDigest,
		app + ":stable":   appDigest,
		other + ":latest": otherDigest,
	} {
		if got := mustResolve(ref); got != want {
			t.Errorf("%q resolves to %s, want %s", ref, got, want)
		}
	}
	for _, bare := range []string{"latest", "stable"} {
		if _, err := resolve(bare); !errors.Is(err, errdef.ErrNotFound) {
			t.Errorf("bare %q lookup = %v, want %v", bare, err, errdef.ErrNotFound)
		}
	}
	if got := run("repo", "tags", "--oci-layout-path", layout, app); got != "latest\nstable\nv1\n" {
		t.Errorf("repository tags = %q, want tags latest, stable, v1", got)
	}

	// The --oci-layout form has no repository: its tags stay bare, including
	// repository-qualified names used by `oras repo tags --oci-layout-path`.
	push("legacy artifact", "--oci-layout", layout+":legacy")
	run("tag", "--oci-layout", layout+":legacy", "legacy-alias", "example.registry.com/foo:latest")
	legacyDigest := mustResolve("legacy")
	for _, alias := range []string{"legacy-alias", "example.registry.com/foo:latest"} {
		if got := mustResolve(alias); got != legacyDigest {
			t.Errorf("%q resolves to %s, want %s", alias, got, legacyDigest)
		}
	}
	// So does a bare reference with --oci-layout-path.
	push("bare artifact", "--oci-layout-path", layout, "bare")
	run("tag", "--oci-layout-path", layout, "bare", "bare-alias")
	if got, want := mustResolve("bare-alias"), mustResolve("bare"); got != want {
		t.Errorf("bare-alias resolves to %s, want %s", got, want)
	}
}

func TestOCILayoutPathTagRejectsInvalidTags(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	layout := filepath.Join(dir, "layout")
	input := filepath.Join(dir, "app.txt")
	app := "localhost:5000/team/app"

	if err := os.WriteFile(input, []byte("artifact"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := New()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{"push", "--oci-layout-path", layout, "--disable-path-validation", app + ":v1", input})
	if err := cmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("push: %v", err)
	}

	var out bytes.Buffer
	tagCmd := New()
	tagCmd.SetOut(&out)
	tagCmd.SetArgs([]string{"tag", "--oci-layout-path", layout, app + ":v1", "example.com/x:y"})
	err := tagCmd.ExecuteContext(ctx)
	if err == nil {
		t.Fatal("tag accepted example.com/x:y, want validation error")
	}
	if !strings.Contains(err.Error(), "invalid extra tag") {
		t.Fatalf("tag error = %v, want invalid extra tag", err)
	}

	store, err := oci.New(layout)
	if err != nil {
		t.Fatal(err)
	}
	malformed := app + ":example.com/x:y"
	if _, err := store.Resolve(ctx, malformed); !errors.Is(err, errdef.ErrNotFound) {
		t.Fatalf("resolve %q = %v, want %v", malformed, err, errdef.ErrNotFound)
	}
}
