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
	stderrs "errors"
	"io"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
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
		Size:      4*1024*1024 + 1,
	}

	_, err := fetchConfigContent(context.Background(), target, desc)
	if !stderrs.Is(err, errdef.ErrSizeExceedsLimit) {
		t.Fatalf("got %v, want %v", err, errdef.ErrSizeExceedsLimit)
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
