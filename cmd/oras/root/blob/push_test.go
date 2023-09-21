//go:build linux || zos || freebsd
// +build linux zos freebsd

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
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/containerd/console"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content/memory"
)

func Test_pushBlobOptions_doPush(t *testing.T) {
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
	var opts pushBlobOptions
	opts.Common.TTY = slave
	// test
	err = opts.doPush(context.Background(), src, desc, r)
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
	if err := orderedMatch(t, buffer.String(), "Uploaded", desc.MediaType, "100.00%", desc.Digest.String()); err != nil {
		t.Fatal(err)
	}
}

func orderedMatch(t *testing.T, actual string, expected ...string) error {
	for _, e := range expected {
		i := strings.Index(actual, e)
		if i < 0 {
			return fmt.Errorf("expected to find %q in %q", e, actual)
		}
		actual = actual[i+len(e):]
	}
	return nil
}
