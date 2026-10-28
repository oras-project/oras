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
	"bytes"
	"context"
	"testing"

	"github.com/oras-project/oras-go/v3"
	"github.com/oras-project/oras-go/v3/content/oci"
)

func Test_showTags_ociLayoutPathWithRepository(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store, err := oci.New(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	desc, err := oras.PackManifest(ctx, store, oras.PackManifestVersion1_1, "application/vnd.test", oras.PackManifestOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, ref := range []string{"localhost:5000/hello:v1", "localhost:5000/hello:v2", "localhost:5000/other:v3"} {
		if err := store.Tag(ctx, desc, ref); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	var out bytes.Buffer
	cmd := showTagsCmd()
	cmd.SetContext(ctx)
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--oci-layout-path", root, "localhost:5000/hello"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("repo tags error = %v", err)
	}
	if got, want := out.String(), "v1\nv2\n"; got != want {
		t.Errorf("repo tags output = %q, want %q", got, want)
	}
}
