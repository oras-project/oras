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

package table

import (
	"bytes"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"oras.land/oras/internal/testutils"
)

func TestTableDiscoverHandler_OneReferrer(t *testing.T) {
	buf := new(bytes.Buffer)
	root := ocispec.Descriptor{Digest: "root"}
	tdh := NewDiscoverHandler(buf, "rawRef", root, false)

	one := ocispec.Descriptor{ArtifactType: ocispec.MediaTypeImageLayer, Digest: "one"}
	if err := tdh.OnDiscovered(one, root); err != nil {
		t.Errorf("OnDiscovered() unexpected error: %v", err)
	}

	if err := tdh.Render(); err != nil {
		t.Errorf("Render() unexpected error: %v", err)
	}
	expected := `Discovered 1 artifact referencing rawRef
Digest: root

Artifact Type                            Digest
application/vnd.oci.image.layer.v1.tar   one
`
	if buf.String() != expected {
		t.Errorf("Expected output <%s> actual <%s>", expected, buf.String())
	}
}

func TestTableDiscoverHandler_NoReferrer(t *testing.T) {
	buf := new(bytes.Buffer)
	root := ocispec.Descriptor{Digest: "root"}
	tdh := NewDiscoverHandler(buf, "rawRef", root, false)

	if err := tdh.Render(); err != nil {
		t.Errorf("Render() unexpected error: %v", err)
	}
	expected := `Discovered 0 artifacts referencing rawRef
Digest: root
`
	if buf.String() != expected {
		t.Errorf("Expected output <%s> actual <%s>", expected, buf.String())
	}
}

func TestTableDiscoverHandler_MultipleReferrer(t *testing.T) {
	buf := new(bytes.Buffer)
	root := ocispec.Descriptor{Digest: "root"}
	tdh := NewDiscoverHandler(buf, "rawRef", root, false)

	one := ocispec.Descriptor{ArtifactType: ocispec.MediaTypeImageLayer, Digest: "one"}
	if err := tdh.OnDiscovered(one, root); err != nil {
		t.Errorf("OnDiscovered() unexpected error: %v", err)
	}

	two := ocispec.Descriptor{ArtifactType: ocispec.MediaTypeImageLayer, Digest: "two"}
	if err := tdh.OnDiscovered(two, root); err != nil {
		t.Errorf("OnDiscovered() unexpected error: %v", err)
	}

	if err := tdh.Render(); err != nil {
		t.Errorf("Render() unexpected error: %v", err)
	}
	expected := `Discovered 2 artifacts referencing rawRef
Digest: root

Artifact Type                            Digest
application/vnd.oci.image.layer.v1.tar   one
application/vnd.oci.image.layer.v1.tar   two
`
	if buf.String() != expected {
		t.Errorf("Expected output <%s> actual <%s>", expected, buf.String())
	}
}

func TestTableDiscoverHandler_Verbose(t *testing.T) {
	buf := new(bytes.Buffer)
	root := ocispec.Descriptor{Digest: "root"}
	tdh := NewDiscoverHandler(buf, "rawRef", root, true)

	one := ocispec.Descriptor{ArtifactType: ocispec.MediaTypeImageLayer, Digest: "one"}
	if err := tdh.OnDiscovered(one, root); err != nil {
		t.Errorf("OnDiscovered() unexpected error: %v", err)
	}

	if err := tdh.Render(); err != nil {
		t.Errorf("Render() unexpected error: %v", err)
	}
	expected := `Discovered 1 artifact referencing rawRef
Digest: root

Artifact Type                            Digest
application/vnd.oci.image.layer.v1.tar   one
{
  "mediaType": "",
  "digest": "one",
  "size": 0,
  "artifactType": "application/vnd.oci.image.layer.v1.tar"
}
`
	if buf.String() != expected {
		t.Errorf("Expected output <%s> actual <%s>", expected, buf.String())
	}
}

func TestTableDiscoverHandler_Failure(t *testing.T) {
	buf := new(bytes.Buffer)
	root := ocispec.Descriptor{Digest: "root"}
	tdh := NewDiscoverHandler(buf, "rawRef", root, false)
	one := ocispec.Descriptor{ArtifactType: ocispec.MediaTypeImageLayer, Digest: "one"}
	notRoot := ocispec.Descriptor{Digest: "notRoot"}

	err := tdh.OnDiscovered(one, notRoot)
	if err == nil {
		t.Fatal("OnDiscovered() Expected error with wrong parameter")
	}
	expected := "unexpected subject descriptor: { notRoot 0 [] map[] [] <nil> }"
	if err.Error() != expected {
		t.Errorf("Expected error <%s> actual error <%s>", expected, err.Error())
	}
}

func Test_discoverHandler_Render(t *testing.T) {
	root := ocispec.Descriptor{Digest: "root"}
	one := ocispec.Descriptor{ArtifactType: ocispec.MediaTypeImageLayer, Digest: "one"}
	two := ocispec.Descriptor{ArtifactType: ocispec.MediaTypeImageLayer, Digest: "two"}

	tests := []struct {
		wf *testutils.WriteFailure
	}{
		{wf: testutils.NewWriteFailure(1)},
		{wf: testutils.NewWriteFailure(2)},
		{wf: testutils.NewWriteFailure(3)},
		{wf: testutils.NewWriteFailure(4)},
		{wf: testutils.NewWriteFailure(5)},
	}
	for _, tt := range tests {
		t.Run(tt.wf.Expected(), func(t *testing.T) {
			tdh := &discoverHandler{
				out:          tt.wf,
				rawReference: "rawRef",
				root:         root,
				verbose:      true,
				referrers:    []ocispec.Descriptor{one, two},
			}
			err := tdh.Render()
			if err == nil {
				t.Errorf("OnDiscovered() Expected error <%s>", tt.wf.Expected())
			} else if err.Error() != tt.wf.Expected() {
				t.Errorf("Expected error <%s> actual error <%s>", tt.wf.Expected(), err.Error())
			}
		})
	}
}
