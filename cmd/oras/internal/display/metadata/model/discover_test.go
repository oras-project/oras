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

package model

import (
	"strings"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

var (
	discoverRoot = ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		Size:      100,
	}
	discoverReferrer = ocispec.Descriptor{
		MediaType:    "application/vnd.oci.image.manifest.v1+json",
		ArtifactType: "application/vnd.example.sbom",
		Digest:       "sha256:2222222222222222222222222222222222222222222222222222222222222222",
		Size:         42,
	}
)

func TestNewNode(t *testing.T) {
	node := NewNode("localhost:5000/test", discoverRoot)

	if want := "localhost:5000/test@" + discoverRoot.Digest.String(); node.Reference != want {
		t.Errorf("NewNode() reference = %q, want %q", node.Reference, want)
	}
	// Referrers starts as an empty slice rather than nil, so it marshals as [].
	if node.Referrers == nil {
		t.Fatal("NewNode() referrers = nil, want an empty slice")
	}
	if len(node.Referrers) != 0 {
		t.Errorf("NewNode() referrers = %v, want empty", node.Referrers)
	}
}

func TestNewDiscover(t *testing.T) {
	d := NewDiscover("localhost:5000/test", discoverRoot)

	if d.Root == nil {
		t.Fatal("NewDiscover() root = nil")
	}
	if want := "localhost:5000/test@" + discoverRoot.Digest.String(); d.Root.Reference != want {
		t.Errorf("NewDiscover() root reference = %q, want %q", d.Root.Reference, want)
	}
}

func TestDiscover_AddReferrer(t *testing.T) {
	d := NewDiscover("localhost:5000/test", discoverRoot)

	if err := d.AddReferrer(discoverReferrer, discoverRoot); err != nil {
		t.Fatalf("AddReferrer() error = %v, want nil", err)
	}
	if len(d.Root.Referrers) != 1 {
		t.Fatalf("AddReferrer() root referrers = %d, want 1", len(d.Root.Referrers))
	}
	if got := d.Root.Referrers[0].Digest; got != discoverReferrer.Digest {
		t.Errorf("AddReferrer() stored digest = %q, want %q", got, discoverReferrer.Digest)
	}
}

// Discovering a referrer makes it a known subject, so a referrer of it attaches
// to that node rather than to the root.
func TestDiscover_AddReferrer_nested(t *testing.T) {
	d := NewDiscover("localhost:5000/test", discoverRoot)
	if err := d.AddReferrer(discoverReferrer, discoverRoot); err != nil {
		t.Fatalf("AddReferrer() error = %v, want nil", err)
	}

	nested := discoverReferrer
	nested.Digest = "sha256:3333333333333333333333333333333333333333333333333333333333333333"
	if err := d.AddReferrer(nested, discoverReferrer); err != nil {
		t.Fatalf("AddReferrer() nested error = %v, want nil", err)
	}

	if len(d.Root.Referrers) != 1 {
		t.Errorf("root referrers = %d, want 1; the nested referrer must not attach to the root", len(d.Root.Referrers))
	}
	if got := len(d.Root.Referrers[0].Referrers); got != 1 {
		t.Errorf("first referrer's referrers = %d, want 1", got)
	}
}

func TestDiscover_AddReferrer_unknownSubject(t *testing.T) {
	d := NewDiscover("localhost:5000/test", discoverRoot)

	unknown := discoverRoot
	unknown.Digest = "sha256:4444444444444444444444444444444444444444444444444444444444444444"

	err := d.AddReferrer(discoverReferrer, unknown)
	if err == nil {
		t.Fatal("AddReferrer() error = nil, want an error for an unknown subject")
	}
	if !strings.Contains(err.Error(), "unexpected subject descriptor") {
		t.Errorf("AddReferrer() error = %q, want it to mention an unexpected subject descriptor", err)
	}
}
