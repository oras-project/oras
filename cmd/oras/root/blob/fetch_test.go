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

package blob

import (
	"bytes"
	"context"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/containerd/console"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content/memory"
)

func Test_fetchBlobOptions_doFetch(t *testing.T) {
	// prepare
	pty, slavePath, err := console.NewPty()
	if err != nil {
		t.Fatal(err)
	}
	slave, err := os.OpenFile(slavePath, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer slave.Close()
	src := memory.New()
	content := []byte("test")
	r := bytes.NewReader(content)
	desc := ocispec.Descriptor{
		MediaType: "application/octet-stream",
		Digest:    digest.FromBytes(content),
		Size:      int64(len(content)),
	}
	tag := "blob"
	ctx := context.Background()
	if err := src.Push(ctx, desc, r); err != nil {
		t.Fatal(err)
	}
	if err := src.Tag(ctx, desc, tag); err != nil {
		t.Fatal(err)
	}

	var opts fetchBlobOptions
	opts.Reference = tag
	opts.Common.TTY = slave
	opts.outputPath = t.TempDir() + "/test"
	// test
	_, err = opts.doFetch(ctx, src)
	if err != nil {
		t.Fatal(err)
	}
	// validate
	var wg sync.WaitGroup
	wg.Add(1)
	var buffer bytes.Buffer
	go func() {
		defer wg.Done()
		_, _ = io.Copy(&buffer, pty)
	}()
	slave.Close()
	wg.Wait()
	if err := orderedMatch(t, buffer.String(), "Downloaded  ", desc.MediaType, "100.00%", desc.Digest.String()); err != nil {
		t.Fatal(err)
	}
}
