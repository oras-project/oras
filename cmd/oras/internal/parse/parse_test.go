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

package parse

import (
	"errors"
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

const manifest = `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"mediaType":"application/vnd.unknown.config.v1+json","digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2},"layers":[{"mediaType":"application/vnd.oci.image.layer.v1.tar","digest":"sha256:5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03","size":6,"annotations":{"org.opencontainers.image.title":"hello.txt"}}]}`
const manifestMediaType = "application/vnd.oci.image.manifest.v1+json"

func Test_MediaTypeFromJson(t *testing.T) {
	// generate test content
	content := []byte(manifest)

	// test MediaTypeFromJson
	want := manifestMediaType
	got, err := MediaTypeFromJson(nil, content)
	if err != nil {
		t.Fatal("ParseMediaType() error=", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseMediaType() = %v, want %v", got, want)
	}
}

func Test_MediaTypeFromJson_invalidContent_notAJson(t *testing.T) {
	// generate test content
	content := []byte("manifest")

	// test MediaTypeFromJson
	_, err := MediaTypeFromJson(nil, content)
	expected := "not a valid json file"
	if err.Error() != expected {
		t.Fatalf("ParseMediaType() error = %v, wantErr %v", err, expected)
	}
}

func Test_MediaTypeFromJson_invalidContent_missingMediaType(t *testing.T) {
	// generate test command
	testParentCmd := &cobra.Command{
		Use: "example parent use",
	}
	testCmd := &cobra.Command{
		Use: "example use",
	}
	testParentCmd.AddCommand(testCmd)

	// generate test content
	content := []byte(`{"schemaVersion":2}`)

	// test MediaTypeFromJson
	_, err := MediaTypeFromJson(testCmd, content)
	if !errors.Is(err, ErrMediaTypeNotFound) {
		t.Fatalf("ParseMediaType() error = %v, wantErr %v", err, ErrMediaTypeNotFound)
	}
}
