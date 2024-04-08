//go:build freebsd || linux || netbsd || openbsd || solaris

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
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras/cmd/oras/internal/display/status/console/testutils"
)

func Test_pushBlobOptions_doPush(t *testing.T) {
	// prepare
	pty, device, err := testutils.NewPty()
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()
	src := memory.New()
	content := []byte("test")
	r := bytes.NewReader(content)
	desc := ocispec.Descriptor{
		MediaType: "application/octet-stream",
		Digest:    digest.FromBytes(content),
		Size:      int64(len(content)),
	}
	var opts pushBlobOptions
	opts.Common.TTY = device
	// test
	err = opts.doPush(context.Background(), src, desc, r)
	if err != nil {
		t.Fatal(err)
	}
	// validate
	if err = testutils.MatchPty(pty, device, "Uploaded", desc.MediaType, "100.00%", desc.Digest.String()); err != nil {
		t.Fatal(err)
	}
}
