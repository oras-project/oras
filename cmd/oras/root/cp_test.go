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
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras/cmd/oras/internal/display/console/testutils"
)

var (
	src  *memory.Store
	desc ocispec.Descriptor
)

func TestMain(m *testing.M) {
	src = memory.New()
	content := []byte("test")
	r := bytes.NewReader(content)
	desc = ocispec.Descriptor{
		MediaType: "application/octet-stream",
		Digest:    digest.FromBytes(content),
		Size:      int64(len(content)),
	}
	if err := src.Push(context.Background(), desc, r); err != nil {
		fmt.Println("Setup failed:", err)
		os.Exit(1)
	}
	if err := src.Tag(context.Background(), desc, desc.Digest.String()); err != nil {
		fmt.Println("Setup failed:", err)
		os.Exit(1)
	}
	m.Run()
}

func Test_doCopy(t *testing.T) {
	// prepare
	pty, slave, err := testutils.NewPty()
	if err != nil {
		t.Fatal(err)
	}
	defer slave.Close()
	var opts copyOptions
	opts.TTY = slave
	opts.Verbose = true
	opts.From.Reference = desc.Digest.String()
	dst := memory.New()
	// test
	_, err = doCopy(context.Background(), src, dst, &opts)
	if err != nil {
		t.Fatal(err)
	}
	// validate
	if err = testutils.MatchPty(pty, slave, "Copied", desc.MediaType, "100.00%", desc.Digest.String()); err != nil {
		t.Fatal(err)
	}
}
