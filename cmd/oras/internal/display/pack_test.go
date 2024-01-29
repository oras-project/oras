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

package display

import (
	"context"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras/cmd/oras/internal/display/console/testutils"
	"oras.land/oras/cmd/oras/internal/display/track"
)

func TestPackHandler_OnCopySkipped(t *testing.T) {
	// prepare
	pty, device, err := testutils.NewPty()
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()
	store := memory.New()
	tty, err := track.NewTarget(store, "", "", device)
	if err != nil {
		t.Fatal(err)
	}
	mockedDigest := "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a"
	mockedDesc := ocispec.Descriptor{
		MediaType: "mockedMediaType",
		Digest:    digest.Digest(mockedDigest),
		Size:      100,
	}
	// test
	ph := NewPackHandler("", nil, nil, tty, true)
	if err = ph.OnCopySkipped(context.Background(), mockedDesc); err != nil {
		t.Fatal(err)
	}
	tty.Close()
	// validate
	if err = testutils.MatchPty(pty, device, mockedDesc.MediaType, "100.00%", mockedDesc.Digest.String()); err != nil {
		t.Fatal(err)
	}
}
