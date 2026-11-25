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
