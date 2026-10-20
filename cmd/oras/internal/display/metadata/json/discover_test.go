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

package json

import (
	"bytes"
	"encoding/json"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// discoverNode mirrors the JSON shape of a node in the discover output.
type discoverNode struct {
	Reference string         `json:"reference"`
	Digest    string         `json:"digest"`
	Referrers []discoverNode `json:"referrers"`
}

func TestDiscoverHandler_Render(t *testing.T) {
	const path = "localhost:5000/test"
	subject := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    testDigest,
		Size:      100,
	}
	signature := ocispec.Descriptor{
		MediaType:    "application/vnd.oci.image.manifest.v1+json",
		Digest:       "sha256:2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae",
		Size:         200,
		ArtifactType: "application/vnd.example.signature",
	}
	// A referrer of the signature, two levels below the subject.
	sbom := ocispec.Descriptor{
		MediaType:    "application/vnd.oci.image.manifest.v1+json",
		Digest:       "sha256:fcde2b2edba56bf408601fb721fe9b5c338d10ee429ea04fae5511b68fbf8fb9",
		Size:         300,
		ArtifactType: "application/vnd.example.sbom",
	}

	buf := &bytes.Buffer{}
	handler := NewDiscoverHandler(buf, subject, path)
	if err := handler.OnDiscovered(signature, subject); err != nil {
		t.Fatalf("OnDiscovered(signature) error = %v, want nil", err)
	}
	if err := handler.OnDiscovered(sbom, signature); err != nil {
		t.Fatalf("OnDiscovered(sbom) error = %v, want nil", err)
	}
	if err := handler.Render(); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	var root discoverNode
	if err := json.Unmarshal(buf.Bytes(), &root); err != nil {
		t.Fatalf("Render() produced invalid JSON: %v", err)
	}
	if want := path + "@" + testDigest; root.Reference != want {
		t.Errorf("root reference = %q, want %q", root.Reference, want)
	}
	if len(root.Referrers) != 1 {
		t.Fatalf("root has %d referrers, want 1", len(root.Referrers))
	}
	if got := root.Referrers[0]; got.Digest != string(signature.Digest) {
		t.Errorf("root referrer digest = %q, want %q", got.Digest, signature.Digest)
	}
	nested := root.Referrers[0].Referrers
	if len(nested) != 1 || nested[0].Digest != string(sbom.Digest) {
		t.Errorf("signature referrers = %+v, want one referrer with digest %q", nested, sbom.Digest)
	}
}

func TestDiscoverHandler_Render_noReferrers(t *testing.T) {
	subject := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    testDigest,
		Size:      100,
	}
	buf := &bytes.Buffer{}
	handler := NewDiscoverHandler(buf, subject, "localhost:5000/test")
	if err := handler.Render(); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	var result map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Render() produced invalid JSON: %v", err)
	}
	if got := string(result["referrers"]); got != "[]" {
		t.Errorf("Render() referrers = %s, want []", got)
	}
}

func TestDiscoverHandler_OnDiscovered_unknownSubject(t *testing.T) {
	subject := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    testDigest,
		Size:      100,
	}
	handler := NewDiscoverHandler(&bytes.Buffer{}, subject, "localhost:5000/test")

	referrer := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    "sha256:2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae",
		Size:      200,
	}
	unknown := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    "sha256:fcde2b2edba56bf408601fb721fe9b5c338d10ee429ea04fae5511b68fbf8fb9",
		Size:      300,
	}
	if err := handler.OnDiscovered(referrer, unknown); err == nil {
		t.Error("OnDiscovered() with a subject that was never discovered: error = nil, want an error")
	}
}
