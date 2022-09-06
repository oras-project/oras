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

package option

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/pflag"
)

func TestDescriptor_ApplyFlags(t *testing.T) {
	var test struct{ Descriptor }
	ApplyFlags(&test, pflag.NewFlagSet("oras-test", pflag.ExitOnError))
	if test.Descriptor.OutputDescriptor != false {
		t.Fatalf("expecting OutputDescriptor to be false but got: %v", test.Descriptor.OutputDescriptor)
	}
}

func TestDescriptor_Marshal(t *testing.T) {
	// generate test content
	blob := []byte("hello world")
	desc := ocispec.Descriptor{
		MediaType: "test",
		Digest:    digest.FromBytes(blob),
		Size:      int64(len(blob)),
	}
	want, err := json.Marshal(desc)
	if err != nil {
		t.Fatal("error calling json.Marshal(), error =", err)
	}

	opts := Descriptor{
		OutputDescriptor: true,
	}
	got, err := opts.Marshal(desc)
	if err != nil {
		t.Fatal("Descriptor.Marshal() error =", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Descriptor.Marshal() got %v, want %v", got, want)
	}
}
