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

package listener

import (
	"context"
	"errors"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote"
)

func TestNewTagListener(t *testing.T) {
	failOnTagging := func(desc ocispec.Descriptor, tag string) error {
		return errors.New("tagging error")
	}
	failOnTagged := func(ocispec.Descriptor, string) error {
		return nil
	}

	target := NewTagListener(memory.New(), failOnTagging, failOnTagged)
	if listened, ok := target.(*tagListenerForTarget); !ok {
		t.Error("expected tagListenerForTarget")
	} else if err := listened.Tag(context.Background(), ocispec.Descriptor{}, "tag"); err == nil {
		t.Error("expecting tagging error but got nil")
	}

	repo, err := remote.NewRepository("oras.land/test:unit-test")
	if err != nil {
		t.Fatal(err)
	}
	target = NewTagListener(repo, failOnTagging, failOnTagged)
	if listened, ok := target.(*tagListenerForRepository); !ok {
		t.Error("expected tagListenerForTarget")
	} else if err := listened.PushReference(context.Background(), ocispec.Descriptor{}, nil, "tag"); err == nil {
		t.Error("expecting tagging error but got nil")
	}
}
