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

package manifest

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/internal/option"
)

func Test_fetchManifest_errType(t *testing.T) {
	// prpare
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	root := t.TempDir()
	err := os.WriteFile(filepath.Join(root, ocispec.ImageLayoutFile), []byte(`{"imageLayoutVersion":"1.0.0"}`), 0644)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	err = os.WriteFile(filepath.Join(root, ocispec.ImageIndexFile), []byte(`{"manifests": [],"mediaType": "application/vnd.oci.image.index.v1+json","schemaVersion": 2}`), 0644)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// test
	opts := &fetchOptions{
		Format: option.Format{
			Type: "unknown",
		},
		Target: option.Target{
			Path:      root,
			Reference: "test",
			Type:      option.TargetTypeOCILayout,
		},
	}
	got := fetchManifest(cmd, opts).Error()
	want := opts.TypeError().Error()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}
