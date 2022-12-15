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
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/pflag"
)

func TestPretty_ApplyFlags(t *testing.T) {
	var test struct{ Pretty }
	ApplyFlags(&test, pflag.NewFlagSet("oras-test", pflag.ExitOnError))
	if test.Pretty.pretty != false {
		t.Fatalf("expecting pretty to be false but got: %v", test.Pretty.pretty)
	}
}

func TestPretty_Output(t *testing.T) {
	// generate test content
	raw := []byte("{\"mediaType\":\"test\",\"digest\":\"sha256:b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9\",\"size\":11}")
	prettified := []byte("{\n  \"mediaType\": \"test\",\n  \"digest\": \"sha256:b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9\",\n  \"size\": 11\n}\n")

	tempDir := t.TempDir()
	fileName := "test.txt"
	path := filepath.Join(tempDir, fileName)
	fp, err := os.Create(path)
	if err != nil {
		t.Fatal("error calling os.Create(), error =", err)
	}
	defer fp.Close()

	// test unprettified content
	opts := Pretty{
		pretty: false,
	}
	err = opts.Output(fp, raw)
	if err != nil {
		t.Fatal("Pretty.Output() error =", err)
	}
	if _, err = fp.Seek(0, io.SeekStart); err != nil {
		t.Fatal("error calling File.Seek(), error =", err)
	}
	got, err := io.ReadAll(fp)
	if err != nil {
		t.Fatal("error calling io.ReadAll(), error =", err)
	}
	if !reflect.DeepEqual(got, raw) {
		t.Fatalf("Pretty.Output() got %v, want %v", got, raw)
	}

	// remove all content in the file
	if err := os.Truncate(path, 0); err != nil {
		t.Fatal("error calling os.Truncate(), error =", err)
	}
	if _, err = fp.Seek(0, io.SeekStart); err != nil {
		t.Fatal("error calling File.Seek(), error =", err)
	}

	// test prettified content
	opts = Pretty{
		pretty: true,
	}
	err = opts.Output(fp, raw)
	if err != nil {
		t.Fatal("Pretty.Output() error =", err)
	}
	if _, err = fp.Seek(0, io.SeekStart); err != nil {
		t.Fatal("error calling File.Seek(), error =", err)
	}
	got, err = io.ReadAll(fp)
	if err != nil {
		t.Fatal("error calling io.ReadAll(), error =", err)
	}

	if !reflect.DeepEqual(got, prettified) {
		t.Fatalf("Pretty.Output() failed to prettified the content: %v", got)
	}
}
