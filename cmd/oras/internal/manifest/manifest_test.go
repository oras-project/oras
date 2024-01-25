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

package manifest

import (
	"errors"
	"reflect"
	"testing"
)

const (
	manifest          = `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"mediaType":"application/vnd.unknown.config.v1+json","digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2},"layers":[{"mediaType":"application/vnd.oci.image.layer.v1.tar","digest":"sha256:5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03","size":6,"annotations":{"org.opencontainers.image.title":"hello.txt"}}]}`
	manifestMediaType = "application/vnd.oci.image.manifest.v1+json"
)

func Test_ExtractMediaType(t *testing.T) {
	// generate test content
	content := []byte(manifest)

	// test ExtractMediaType
	want := manifestMediaType
	got, err := ExtractMediaType(content)
	if err != nil {
		t.Fatal("ExtractMediaType() error=", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ExtractMediaType() = %v, want %v", got, want)
	}
}

func Test_ExtractMediaType_invalidContent_notAJson(t *testing.T) {
	// generate test content
	content := []byte("manifest")

	// test ExtractMediaType
	_, err := ExtractMediaType(content)
	expected := "not a valid json file"
	if err.Error() != expected {
		t.Fatalf("ExtractMediaType() error = %v, wantErr %v", err, expected)
	}
}

func Test_ExtractMediaType_invalidContent_missingMediaType(t *testing.T) {
	// generate test content
	content := []byte(`{"schemaVersion":2}`)

	// test ExtractMediaType
	_, err := ExtractMediaType(content)
	if !errors.Is(err, ErrMediaTypeNotFound) {
		t.Fatalf("ExtractMediaType() error = %v, wantErr %v", err, ErrMediaTypeNotFound)
	}
}
