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
)

func TestNewTagListener(t *testing.T) {
	target := NewTagListener(memory.New(), func(desc ocispec.Descriptor, tag string) error {
		return errors.New("tagging error")
	}, func(ocispec.Descriptor, string) error { return nil })
	if listened, ok := target.(*tagListenerForTarget); !ok {
		t.Error("expected tagListenerForTarget")
	} else if err := listened.Tag(context.Background(), ocispec.Descriptor{}, "tag"); err == nil {
		t.Error("expecting tagging error but got nil")
	}
}
