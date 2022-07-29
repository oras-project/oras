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

package cache

import (
	"bytes"
	"context"
	_ "crypto/sha256"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
)

func TestProxyCache(t *testing.T) {
	blob := []byte("hello world")
	desc := ocispec.Descriptor{
		MediaType: "test",
		Digest:    digest.FromBytes(blob),
		Size:      int64(len(blob)),
	}

	p := New(memory.New(), memory.New())
	ctx := context.Background()

	err := p.Push(ctx, desc, bytes.NewReader(blob))
	if err != nil {
		t.Fatal("Proxy.Push() error =", err)
	}

	// first fetch
	exists, err := p.Exists(ctx, desc)
	if err != nil {
		t.Fatal("Proxy.Exists() error =", err)
	}
	if !exists {
		t.Errorf("Proxy.Exists() = %v, want %v", exists, true)
	}
	got, err := content.FetchAll(ctx, p, desc)
	if err != nil {
		t.Fatal("Proxy.Fetch() error =", err)
	}
	if !bytes.Equal(got, blob) {
		t.Errorf("Proxy.Fetch() = %v, want %v", got, blob)
	}

	// repeated fetch should not touch base CAS
	// nil base will generate panic if the base CAS is touched
	p.(*proxy).Target = nil

	exists, err = p.Exists(ctx, desc)
	if err != nil {
		t.Fatal("Proxy.Exists() error =", err)
	}
	if !exists {
		t.Errorf("Proxy.Exists() = %v, want %v", exists, true)
	}
	got, err = content.FetchAll(ctx, p, desc)
	if err != nil {
		t.Fatal("Proxy.Fetch() error =", err)
	}
	if !bytes.Equal(got, blob) {
		t.Errorf("Proxy.Fetch() = %v, want %v", got, blob)
	}
}

func TestProxyPushPassThrough(t *testing.T) {
	blob := []byte("hello world")
	desc := ocispec.Descriptor{
		MediaType: "test",
		Digest:    digest.FromBytes(blob),
		Size:      int64(len(blob)),
	}

	p := New(memory.New(), memory.New())
	ctx := context.Background()

	// before push
	exists, err := p.Exists(ctx, desc)
	if err != nil {
		t.Fatal("Proxy.Exists() error =", err)
	}
	if exists {
		t.Errorf("Proxy.Exists() = %v, want %v", exists, false)
	}

	// push
	err = p.Push(ctx, desc, bytes.NewReader(blob))
	if err != nil {
		t.Fatal("Proxy.Push() error =", err)
	}

	// after push
	exists, err = p.Exists(ctx, desc)
	if err != nil {
		t.Fatal("Proxy.Exists() error =", err)
	}
	if !exists {
		t.Errorf("Proxy.Exists() = %v, want %v", exists, true)
	}
	got, err := content.FetchAll(ctx, p, desc)
	if err != nil {
		t.Fatal("Proxy.Fetch() error =", err)
	}
	if !bytes.Equal(got, blob) {
		t.Errorf("Proxy.Fetch() = %v, want %v", got, blob)
	}
}
