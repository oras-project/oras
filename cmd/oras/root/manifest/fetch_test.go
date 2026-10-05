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
	"bytes"
	"context"
	"encoding/json"
	stderrs "errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/root/manifest/common"
)

type testConfigTarget struct {
	content []byte
}

func (t *testConfigTarget) Exists(context.Context, ocispec.Descriptor) (bool, error) {
	return false, nil
}

func (t *testConfigTarget) Fetch(context.Context, ocispec.Descriptor) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(t.content)), nil
}

func (t *testConfigTarget) Resolve(context.Context, string) (ocispec.Descriptor, error) {
	return ocispec.Descriptor{}, nil
}

var _ oras.ReadOnlyTarget = (*testConfigTarget)(nil)

func Test_fetchManifest_errType(t *testing.T) {
	// prepare
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	// test
	opts := &fetchOptions{
		Format: option.Format{
			Type: "unknown",
		},
	}
	got := fetchManifest(cmd, opts).Error()
	want := errors.UnsupportedFormatTypeError(opts.Format.Type).Error()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func Test_fetchCmd_outputAndFormat(t *testing.T) {
	cmd := fetchCmd()
	cmd.SetArgs([]string{
		"localhost:5000/hello:v1",
		"--output", "-",
		"--format", "go-template",
		"--template", "{{.digest}}",
	})

	err := cmd.Execute()
	want := "`--output -` cannot be used with `--format go-template` at the same time"
	if err == nil || err.Error() != want {
		t.Fatalf("got %v, want %v", err, want)
	}
}

func Test_fetchConfigContent_sizeExceedsLimit(t *testing.T) {
	target := &testConfigTarget{}
	desc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageConfig,
		Size:      common.MaxConfigSize + 1,
	}

	_, err := fetchConfigContent(context.Background(), target, desc)
	if !stderrs.Is(err, errdef.ErrSizeExceedsLimit) {
		t.Fatalf("got %v, want %v", err, errdef.ErrSizeExceedsLimit)
	}
}

func Test_fetchConfig_outputStdout_exceedsLimit(t *testing.T) {
	tempDir := t.TempDir()
	layoutDir := filepath.Join(tempDir, "layout")

	config := bytes.Repeat([]byte("a"), int(common.MaxConfigSize)+1)
	configDigest := digest.FromBytes(config)

	manifest := []byte(fmt.Sprintf(`{
		"schemaVersion": 2,
		"config": {
			"mediaType": %q,
			"digest": %q,
			"size": %d
		},
		"layers": []
	}`, ocispec.MediaTypeImageConfig, configDigest, len(config)))
	manifestDigest := digest.FromBytes(manifest)

	if err := os.MkdirAll(filepath.Join(layoutDir, "blobs", "sha256"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(layoutDir, "oci-layout"),
		[]byte(`{"imageLayoutVersion":"1.0.0"}`),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	index := []byte(fmt.Sprintf(`{
		"schemaVersion": 2,
		"manifests": [{
			"mediaType": %q,
			"digest": %q,
			"size": %d,
			"annotations": {
				"org.opencontainers.image.ref.name": "test:v1"
			}
		}]
	}`, ocispec.MediaTypeImageManifest, manifestDigest, len(manifest)))

	if err := os.WriteFile(
		filepath.Join(layoutDir, "index.json"),
		index,
		0644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(layoutDir, "blobs", "sha256", configDigest.Encoded()),
		config,
		0644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(layoutDir, "blobs", "sha256", manifestDigest.Encoded()),
		manifest,
		0644,
	); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer

	rootCmd := &cobra.Command{Use: "oras"}
	cmd := fetchConfigCmd()
	cmd.SetOut(&output)
	rootCmd.AddCommand(cmd)
	rootCmd.SetArgs([]string{
		"fetch-config",
		"--oci-layout-path", layoutDir,
		"test:v1",
		"--output", "-",
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(output.Bytes(), config) {
		t.Fatalf("got %d bytes, want %d", output.Len(), len(config))
	}
}

func Test_fetchConfigContent_withinLimit(t *testing.T) {
	data := []byte(`{"architecture":"amd64"}`)
	target := &testConfigTarget{
		content: data,
	}
	desc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageConfig,
		Digest:    digest.FromBytes(data),
		Size:      int64(len(data)),
	}

	got, err := fetchConfigContent(context.Background(), target, desc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("got %q, want %q", got, data)
	}
}

func Test_fetchConfig_outputFile(t *testing.T) {
	tempDir := t.TempDir()
	layoutDir := filepath.Join(tempDir, "layout")
	outputPath := filepath.Join(tempDir, "config.json")

	config := []byte(`{"architecture":"amd64"}`)
	configDigest := digest.FromBytes(config)

	manifest := []byte(fmt.Sprintf(`{
		"schemaVersion": 2,
		"config": {
			"mediaType": %q,
			"digest": %q,
			"size": %d
		},
		"layers": []
	}`, ocispec.MediaTypeImageConfig, configDigest, len(config)))
	manifestDigest := digest.FromBytes(manifest)

	if err := os.MkdirAll(filepath.Join(layoutDir, "blobs", "sha256"), 0755); err != nil {
		t.Fatal(err)
	}

	ociLayout := []byte(`{"imageLayoutVersion":"1.0.0"}`)
	if err := os.WriteFile(filepath.Join(layoutDir, "oci-layout"), ociLayout, 0644); err != nil {
		t.Fatal(err)
	}

	index := []byte(fmt.Sprintf(`{
		"schemaVersion": 2,
		"manifests": [{
			"mediaType": %q,
			"digest": %q,
			"size": %d,
			"annotations": {
				"org.opencontainers.image.ref.name": "test:v1"
			}
		}]
	}`, ocispec.MediaTypeImageManifest, manifestDigest, len(manifest)))
	if err := os.WriteFile(filepath.Join(layoutDir, "index.json"), index, 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(layoutDir, "blobs", "sha256", configDigest.Encoded()),
		config,
		0644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(layoutDir, "blobs", "sha256", manifestDigest.Encoded()),
		manifest,
		0644,
	); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer

	rootCmd := &cobra.Command{Use: "oras"}
	cmd := fetchConfigCmd()
	cmd.SetOut(&output)
	rootCmd.AddCommand(cmd)
	rootCmd.SetArgs([]string{
		"fetch-config",
		"--oci-layout-path", layoutDir,
		"test:v1",
		"--output", outputPath,
		"--descriptor",
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, config) {
		t.Fatalf("got %q, want %q", got, config)
	}
	var gotDesc ocispec.Descriptor
	if err := json.Unmarshal(output.Bytes(), &gotDesc); err != nil {
		t.Fatalf("failed to decode descriptor output: %v", err)
	}
	if gotDesc.Digest != configDigest {
		t.Fatalf("got digest %q, want %q", gotDesc.Digest, configDigest)
	}
}

func Test_fetchConfig_outputFile_fetchError(t *testing.T) {
	tempDir := t.TempDir()
	layoutDir := filepath.Join(tempDir, "layout")
	outputPath := filepath.Join(tempDir, "config.json")

	configDigest := digest.FromString("missing-config")

	manifest := []byte(fmt.Sprintf(`{
		"schemaVersion": 2,
		"config": {
			"mediaType": %q,
			"digest": %q,
			"size": 100
		},
		"layers": []
	}`, ocispec.MediaTypeImageConfig, configDigest))
	manifestDigest := digest.FromBytes(manifest)

	if err := os.MkdirAll(filepath.Join(layoutDir, "blobs", "sha256"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(layoutDir, "oci-layout"),
		[]byte(`{"imageLayoutVersion":"1.0.0"}`),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	index := []byte(fmt.Sprintf(`{
		"schemaVersion": 2,
		"manifests": [{
			"mediaType": %q,
			"digest": %q,
			"size": %d,
			"annotations": {
				"org.opencontainers.image.ref.name": "test:v1"
			}
		}]
	}`, ocispec.MediaTypeImageManifest, manifestDigest, len(manifest)))

	if err := os.WriteFile(
		filepath.Join(layoutDir, "index.json"),
		index,
		0644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(layoutDir, "blobs", "sha256", manifestDigest.Encoded()),
		manifest,
		0644,
	); err != nil {
		t.Fatal(err)
	}

	rootCmd := &cobra.Command{Use: "oras"}
	cmd := fetchConfigCmd()
	rootCmd.AddCommand(cmd)
	rootCmd.SetArgs([]string{
		"fetch-config",
		"--oci-layout-path", layoutDir,
		"test:v1",
		"--output", outputPath,
	})

	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected fetch error")
	}

	if _, err := os.Stat(outputPath); err == nil {
		t.Fatal("output file was created after fetch failed")
	} else if !os.IsNotExist(err) {
		t.Fatalf("failed to check output file: %v", err)
	}
}

func Test_fetchConfig_outputFile_createError(t *testing.T) {
	tempDir := t.TempDir()
	layoutDir := filepath.Join(tempDir, "layout")
	outputPath := filepath.Join(tempDir, "missing", "config.json")

	config := []byte(`{"architecture":"amd64"}`)
	configDigest := digest.FromBytes(config)

	manifest := []byte(fmt.Sprintf(`{
		"schemaVersion": 2,
		"config": {
			"mediaType": %q,
			"digest": %q,
			"size": %d
		},
		"layers": []
	}`, ocispec.MediaTypeImageConfig, configDigest, len(config)))
	manifestDigest := digest.FromBytes(manifest)

	if err := os.MkdirAll(filepath.Join(layoutDir, "blobs", "sha256"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(layoutDir, "oci-layout"), []byte(`{"imageLayoutVersion":"1.0.0"}`), 0644); err != nil {
		t.Fatal(err)
	}

	index := []byte(fmt.Sprintf(`{
		"schemaVersion": 2,
		"manifests": [{
			"mediaType": %q,
			"digest": %q,
			"size": %d,
			"annotations": {
				"org.opencontainers.image.ref.name": "test:v1"
			}
		}]
	}`, ocispec.MediaTypeImageManifest, manifestDigest, len(manifest)))

	if err := os.WriteFile(filepath.Join(layoutDir, "index.json"), index, 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(layoutDir, "blobs", "sha256", configDigest.Encoded()),
		config,
		0644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(layoutDir, "blobs", "sha256", manifestDigest.Encoded()),
		manifest,
		0644,
	); err != nil {
		t.Fatal(err)
	}

	rootCmd := &cobra.Command{Use: "oras"}
	cmd := fetchConfigCmd()
	rootCmd.AddCommand(cmd)
	rootCmd.SetArgs([]string{
		"fetch-config",
		"--oci-layout-path", layoutDir,
		"test:v1",
		"--output", outputPath,
	})

	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected output file creation error")
	}
}

func Test_fetchConfig_outputFile_digestMismatch(t *testing.T) {
	tempDir := t.TempDir()
	layoutDir := filepath.Join(tempDir, "layout")
	outputPath := filepath.Join(tempDir, "config.json")

	original := []byte("original config")
	if err := os.WriteFile(outputPath, original, 0644); err != nil {
		t.Fatal(err)
	}

	config := []byte(`{"architecture":"amd64"}`)
	expectedDigest := digest.FromString("expected")

	manifest := []byte(fmt.Sprintf(`{
		"schemaVersion": 2,
		"config": {
			"mediaType": %q,
			"digest": %q,
			"size": %d
		},
		"layers": []
	}`, ocispec.MediaTypeImageConfig, expectedDigest, len(config)))
	manifestDigest := digest.FromBytes(manifest)

	if err := os.MkdirAll(filepath.Join(layoutDir, "blobs", "sha256"), 0755); err != nil {
		t.Fatal(err)
	}

	ociLayout := []byte(`{"imageLayoutVersion":"1.0.0"}`)
	if err := os.WriteFile(filepath.Join(layoutDir, "oci-layout"), ociLayout, 0644); err != nil {
		t.Fatal(err)
	}

	index := []byte(fmt.Sprintf(`{
		"schemaVersion": 2,
		"manifests": [{
			"mediaType": %q,
			"digest": %q,
			"size": %d,
			"annotations": {
				"org.opencontainers.image.ref.name": "test:v1"
			}
		}]
	}`, ocispec.MediaTypeImageManifest, manifestDigest, len(manifest)))
	if err := os.WriteFile(filepath.Join(layoutDir, "index.json"), index, 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(layoutDir, "blobs", "sha256", expectedDigest.Encoded()),
		config,
		0644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(layoutDir, "blobs", "sha256", manifestDigest.Encoded()),
		manifest,
		0644,
	); err != nil {
		t.Fatal(err)
	}

	rootCmd := &cobra.Command{Use: "oras"}
	cmd := fetchConfigCmd()
	rootCmd.AddCommand(cmd)
	rootCmd.SetArgs([]string{
		"fetch-config",
		"--oci-layout-path", layoutDir,
		"test:v1",
		"--output", outputPath,
	})

	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected digest mismatch error")
	} else if !strings.Contains(err.Error(), "mismatched digest") {
		t.Fatalf("got %v, want mismatched digest error", err)
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("output file was modified: got %q, want %q", got, original)
	}
}
