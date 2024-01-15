package parse

import (
	"reflect"
	"testing"

	"github.com/spf13/cobra"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
)

const manifest = `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"mediaType":"application/vnd.unknown.config.v1+json","digest":"sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a","size":2},"layers":[{"mediaType":"application/vnd.oci.image.layer.v1.tar","digest":"sha256:5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03","size":6,"annotations":{"org.opencontainers.image.title":"hello.txt"}}]}`
const manifestMediaType = "application/vnd.oci.image.manifest.v1+json"

func Test_MediaTypeFromJson(t *testing.T) {
	// generate test content
	content := []byte(manifest)

	// test ParseMediaType
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

	// test ParseMediaType
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

	// test ParseMediaType
	_, err := MediaTypeFromJson(testCmd, content)
	expected := ErrMediaTypeNotFound
	gotError, isOrasError := err.(*oerrors.Error)
	if !isOrasError {
		t.Fatal("incorrect error type, expect *oerrors.Error")
	}
	if gotError.Unwrap() != expected {
		t.Fatalf("ParseMediaType() error = %v, wantErr %v", err, expected)
	}
}
