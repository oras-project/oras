//go:build darwin || freebsd || linux || netbsd || openbsd || solaris

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
	"context"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras/cmd/oras/internal/display/console/testutils"
)

var (
	memStore *memory.Store
	desc     ocispec.Descriptor
)

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
	_, err = doCopy(context.Background(), memStore, dst, opts)
	if err != nil {
		t.Fatal(err)
	}
	// validate
	if err = testutils.MatchPty(pty, slave, "Copied", desc.MediaType, "100.00%", desc.Digest.String()); err != nil {
		t.Fatal(err)
	}
}

func Test_doCopy_skipped(t *testing.T) {
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
	// test
	_, err = doCopy(context.Background(), memStore, memStore, opts)
	if err != nil {
		t.Fatal(err)
	}
	// validate
	if err = testutils.MatchPty(pty, slave, "Exists", desc.MediaType, "100.00%", desc.Digest.String()); err != nil {
		t.Fatal(err)
	}
}
